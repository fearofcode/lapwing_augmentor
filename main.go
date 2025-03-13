package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"slices"
	"sort"
	"strings"
)

const (
	properNameStrokeLengthLimit = 8
)

func sortedMapKeys[V string | []string](dict *map[string]V) []string {
	keys := make([]string, 0, len(*dict))
	for key := range *dict {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	// sort by length, then lexicographically, so more common words generally come first
	// and we don't need comprehensive word usage data to sort by commonness
	slices.SortFunc(keys, func(a, b string) int {
		if len(a) != len(b) {
			return cmp.Compare(len(a), len(b))
		} else {
			return strings.Compare(a, b)
		}
	})
	return keys
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ", ")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

func main() {

	logger := log.New(os.Stdout, "LOG: ", log.LstdFlags|log.Lmicroseconds)
	var (
		sourceDictPaths stringList
		targetDictPaths stringList
	)
	flag.Var(&sourceDictPaths, "lapwing_source", "source dictionary path(s)")
	flag.Var(&targetDictPaths, "output_target", "target dictionary path(s)")
	flag.Parse()

	if len(sourceDictPaths) == 0 || len(targetDictPaths) == 0 {
		fmt.Println("Usage: lapwing_augmentor --lapwing_source <source-dict> [--lapwing_source <source-dict2> ...] --output_target <target-dict> [--output_target <target-dict2> ...]")
		os.Exit(1)
	}

	logger.Println("Reading in dictionary from ", sourceDictPaths)
	originalDictionary := make(map[string]string)

	for _, sourceDictPath := range sourceDictPaths {
		sourceContents, err := os.ReadFile(sourceDictPath)
		if err != nil {
			fmt.Println("Error reading source dictionary:", err)
			os.Exit(1)
		}
		var sourceJSON map[string]interface{}

		if err := json.Unmarshal([]byte(sourceContents), &sourceJSON); err != nil {
			fmt.Println("Error parsing JSON:", err)
			os.Exit(1)
		}
		for key, value := range sourceJSON {
			originalDictionary[key] = value.(string)
		}
	}

	logger.Println("Done reading in dictionary(s). Combined size:", len(originalDictionary))

	prefixTree := NewPrefixTree()

	logger.Println("Populating prefix tree")

	for key := range originalDictionary {
		prefixTree.Insert(strings.Split(key, "/"))
	}

	logger.Println("Done populating prefix tree")

	additionalEntries := make(map[string]string)
	kwrSuffixPattern := `^.*/KWR([^/]+)$`
	kwrSuffixRegex := regexp.MustCompile(kwrSuffixPattern)
	suffixReplacements := make(map[string][]string)
	directReplacementSuffixPairs := []string{
		"H",
		"HR",
		"K",
		"KH",
		"KR",
		"KW",
		"P",
		"PH",
		"PW",
		"R",
		"S",
		"SR",
		"SKWR",
		"T",
		"TH",
		"THR",
		"TK",
		"TKPW",
		"TPH",
		"TP",
		"TR",
		"W",
	}
	for _, suffix := range directReplacementSuffixPairs {
		suffixReplacements["/"+suffix+"EU"] = []string{"/" + suffix + "AOE", "/" + suffix + "AE"}
	}
	suffixReplacements["/-B/KWREU"] = []string{"/PWEU"}
	suffixReplacements["/-BL/KWREU"] = []string{"/PWHREU"}
	suffixReplacements["/-FL/KWREU"] = []string{"/TPHREU"}
	suffixReplacements["/-L/KWREU"] = []string{"/HREU"}
	suffixReplacements["/-P/KWREU"] = []string{"/PEU"}
	suffixReplacements["/-PL/KWREU"] = []string{"/PHREU"}
	suffixReplacements["R/KWREU"] = []string{"/REU"}
	suffixReplacements["PB/KWREU"] = []string{"/TPHEU"}
	suffixReplacements["PL/KWREU"] = []string{"/PHEU"}
	suffixReplacements["F/KWREU"] = []string{"/TPEU"}
	suffixReplacements["BG/KWREU"] = []string{"/KEU"}
	suffixReplacements["S"] = []string{"Z"}
	suffixReplacementKeys := sortedMapKeys(&suffixReplacements)
	stringReplacements := make(map[string][]string)
	stringReplacements["/-B/KWR"] = []string{"/PW"}
	stringReplacements["/-BL/KWR"] = []string{"/PWHR"}
	stringReplacements["/-FL/KWR"] = []string{"/TPHR"}
	stringReplacements["/-L/KWR"] = []string{"/HR"}
	stringReplacements["/-P/KWR"] = []string{"/P"}
	stringReplacements["/-PL/KWR"] = []string{"/PHR"}
	stringReplacements["D/KWR"] = []string{"/TK"}      // D
	stringReplacements["G/KWR"] = []string{"/TPKW"}    // G
	stringReplacements["PBLG/KWR"] = []string{"/PBLG"} // J
	stringReplacements["BG/KWR"] = []string{"/K"}      // K
	stringReplacements["L/KWR"] = []string{"/PBLG"}    // L
	stringReplacements["PL/KWR"] = []string{"/PH"}     // M
	stringReplacements["PB/KWR"] = []string{"/TPH"}    // N
	stringReplacements["P/KWR"] = []string{"/P"}       // P
	stringReplacements["R/KWR"] = []string{"/R"}       // R
	stringReplacements["S/KWR"] = []string{"/S"}       // S
	stringReplacements["T/KWR"] = []string{"/T"}       // T
	stringReplacements["Z/KWR"] = []string{"/STKPW"}   // Z
	stringReplacements["STKPW"] = []string{"Z"}        // Z
	stringReplacements["SR"] = []string{"V"}           // Z
	stringReplacementKeys := sortedMapKeys(&stringReplacements)

	vowelsDashes := `[AEOU\-*]+`
	vowelDashRegex := regexp.MustCompile(vowelsDashes)
	rightHandAfterS := regexp.MustCompile(`[DZ]`)
	originalDictionaryIndex := 0
	sortedOriginalDictionaryKeys := sortedMapKeys(&originalDictionary)
	for _, key := range sortedOriginalDictionaryKeys {
		value := originalDictionary[key]
		originalDictionaryIndex++
		if originalDictionaryIndex%10000 == 0 {
			logger.Println("Processed", originalDictionaryIndex, "/", len(originalDictionary), "entries")
		}

		strokes := strings.Split(key, "/")
		if len(strokes) > properNameStrokeLengthLimit && value[0] >= 'A' && value[0] <= 'Z' {
			logger.Println("Skipping key", key, "value = ", value, "since it looks to be a proper name with > ",
				properNameStrokeLengthLimit, " strokes and probably has no strokes worth generating")
			continue
		}
		if len(strokes) >= 2 {
			alternateStrokes := generateAlternateSyllableSplitStrokes(strokes, &originalDictionary, &additionalEntries, prefixTree)
			for _, strokeSet := range alternateStrokes {
				addEntryIfNotPresent(strings.Join(strokeSet, "/"), value, &originalDictionary, &additionalEntries, prefixTree)
			}

			// look for cases where we can safely remove KWR without creating word boundary errors
			if strings.Contains(key, "/KWR") {
				variations := generateKwrRemovedVariations(key, strokes, &originalDictionary)
				for _, variation := range variations {
					addEntryIfNotPresent(strings.Join(variation, "/"), value, &originalDictionary, &additionalEntries, prefixTree)
				}
			}
		}

		generateSZVariationForKey(key, strokes, vowelDashRegex, rightHandAfterS, value, &originalDictionary, &additionalEntries, prefixTree)

		addSuffixReplacements(suffixReplacementKeys, suffixReplacements, key, value, &originalDictionary, &additionalEntries, prefixTree)
		addStringReplacements(stringReplacementKeys, stringReplacements, key, value, &originalDictionary, &additionalEntries, prefixTree)

		// for strokes that end with e.g. "/-<letters>", see if we can fold that into the last stroke
		lastStroke := strokes[len(strokes)-1]
		if strings.HasPrefix(lastStroke, "-") {
			newStroke := strings.Replace(lastStroke, "-", "", 1)
			newKey := strings.TrimSuffix(key, "/"+lastStroke) + newStroke
			// this will check if it's a valid steno stroke
			addEntryIfNotPresent(newKey, value, &originalDictionary, &additionalEntries, prefixTree)
			// now see if we can also fold in S/Z
			keyStrokes := strings.Split(newKey, "/")
			generateSZVariationForKey(newKey, keyStrokes, vowelDashRegex, rightHandAfterS, value, &originalDictionary, &additionalEntries, prefixTree)
		}
		kwrMatch := kwrSuffixRegex.FindStringSubmatch(key)
		if kwrMatch != nil {
			kwrSuffix := kwrMatch[1]
			kwrPrefix := strings.TrimSuffix(key, kwrSuffix)
			if kwrSuffix == "" {
				log.Println("No KWR suffix found in key:", key, "value:", value)
				continue
			}
			// act on KWREU cases, but skip cases like lefty-loosy and hanky-panky
			// so that we don't mix KWREU and KWRAE/AOE in the same outline which is kind of confusing
			if kwrSuffix == "EU" && !(strings.Contains(key, "/KWREU/") && (strings.Contains(value, "y-") || strings.Contains(value, "y "))) {
				keyVariation1 := fmt.Sprintf("%sAOE", kwrPrefix)
				addEntryIfNotPresent(keyVariation1, value, &originalDictionary, &additionalEntries, prefixTree)
				keyVariation2 := fmt.Sprintf("%sAE", kwrPrefix)
				addEntryIfNotPresent(keyVariation2, value, &originalDictionary, &additionalEntries, prefixTree)
			}
		}
	}

	additionalEntryIndex := 0
	sortedAdditionalEntryKeys := sortedMapKeys(&additionalEntries)
	for _, key := range sortedAdditionalEntryKeys {
		value := additionalEntries[key]
		additionalEntryIndex++
		strokes := strings.Split(key, "/")
		if len(strokes) >= 2 {
			// see if we can generate KWR removed variations on additional entries we just generated
			if strings.Contains(key, "/KWR") {

				variations := generateKwrRemovedVariations(key, strokes, &originalDictionary)
				for _, variation := range variations {
					addEntryIfNotPresent(strings.Join(variation, "/"), value, &originalDictionary, &additionalEntries, prefixTree)
				}
			}
		}
		// see if we can generate suffix variations of generated additional entries
		addSuffixReplacements(suffixReplacementKeys, suffixReplacements, key, value, &originalDictionary, &additionalEntries, prefixTree)
		addStringReplacements(stringReplacementKeys, stringReplacements, key, value, &originalDictionary, &additionalEntries, prefixTree)
	}

	// one last time
	additionalEntryIndex = 0
	sortedAdditionalEntryKeys = sortedMapKeys(&additionalEntries)
	for _, key := range sortedAdditionalEntryKeys {
		value := additionalEntries[key]
		additionalEntryIndex++
		if additionalEntryIndex%1000 == 0 {
			logger.Println("Processed", additionalEntryIndex, "/", len(sortedAdditionalEntryKeys), "additional entries (alternate splits)")
		}
		strokes := strings.Split(key, "/")
		if len(strokes) >= 2 {
			// now try generating alternate syllabic splits on previously added entries
			alternateStrokes := generateAlternateSyllableSplitStrokes(strokes, &originalDictionary, &additionalEntries, prefixTree)
			for _, strokeSet := range alternateStrokes {
				addEntryIfNotPresent(strings.Join(strokeSet, "/"), value, &originalDictionary, &additionalEntries, prefixTree)
			}
		}
	}

	// try to find words where we can add KWR in places we generated alternate splits
	// in case KWR is being used for silent linker
	// this comes about when we take a word like "synovia" which lapwing has as SEU/TPOEF/KWRA
	// we move the TP over to the right hand to give SEUB/OEF/KWRA which is fine but we should also generate SEUB/KWROEF/KWRA
	additionalEntryIndex = 0
	sortedAdditionalEntryKeys = sortedMapKeys(&additionalEntries)
	for _, key := range sortedAdditionalEntryKeys {
		additionalEntryIndex++
		if additionalEntryIndex%1000 == 0 {
			logger.Println("Processed", additionalEntryIndex, "/", len(sortedAdditionalEntryKeys), "additional entries (KWR addition)")
		}
		strokes := strings.Split(key, "/")
		if len(strokes) >= 2 {
			kwrAddedStrokes := make([]string, len(strokes))
			copy(kwrAddedStrokes, strokes)
			for i, stroke := range strokes {
				startsWithVowel := strings.HasPrefix(stroke, "A") || strings.HasPrefix(stroke, "E") ||
					strings.HasPrefix(stroke, "O") || strings.HasPrefix(stroke, "U")
				// only replace second stroke or later
				if i > 0 && startsWithVowel {
					kwrAddedStrokes[i] = "KWR" + kwrAddedStrokes[i]
				}
			}
			addEntryIfNotPresent(strings.Join(kwrAddedStrokes, "/"), additionalEntries[key], &originalDictionary, &additionalEntries, prefixTree)
		}
	}

	// do a final check of additional entries for valid word boundaries due to weird issues with order of addition
	additionalEntryIndex = 0
	sortedAdditionalEntryKeys = sortedMapKeys(&additionalEntries)
	for _, key := range sortedAdditionalEntryKeys {
		additionalEntryIndex++
		if additionalEntryIndex%10000 == 0 {
			logger.Println("Processed", additionalEntryIndex, "/", len(sortedAdditionalEntryKeys), "additional entries for final conflicting word boundaries")
		}
		strokes := strings.Split(key, "/")
		if len(strokes) >= 2 {
			if !validWordBoundaries(strokes, &originalDictionary, &additionalEntries, prefixTree) {
				logger.Println("Removing", key, "due to conflicting word boundaries")
				delete(additionalEntries, key)
			}
		}
	}
	log.Println("Added", len(additionalEntries), "additional entries overall after checking for conflicting word boundaries")

	// write out additionalEntries to file at targetDictPath
	contents, err := json.MarshalIndent(additionalEntries, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		os.Exit(1)
	}
	for _, targetPath := range targetDictPaths {
		if err := os.WriteFile(targetPath, contents, 0644); err != nil {
			fmt.Println("Error writing to target dictionary:", err)
			os.Exit(1)
		}
		log.Println("Wrote", len(additionalEntries), "additional entries to", targetPath)
	}

}

func addSuffixReplacements(suffixReplacementKeys []string, suffixReplacements map[string][]string, key string, value string, originalDictionary *map[string]string,
	additionalEntries *map[string]string, prefixTree *PrefixTree) {
	for _, replacedSuffix := range suffixReplacementKeys {
		replacements := suffixReplacements[replacedSuffix]
		if strings.HasSuffix(key, replacedSuffix) {
			for _, replacement := range replacements {
				newKey := strings.TrimSuffix(key, replacedSuffix) + replacement
				newKey = strings.ReplaceAll(newKey, "//", "/")
				addEntryIfNotPresent(newKey, value, originalDictionary, additionalEntries, prefixTree)
			}
			break
		}
	}
}

func addStringReplacements(replacementKeys []string, replacements map[string][]string, key string, value string, originalDictionary *map[string]string,
	additionalEntries *map[string]string, prefixTree *PrefixTree) {
	for _, replacedKey := range replacementKeys {
		replacements := replacements[replacedKey]
		if strings.Contains(key, replacedKey) {
			for _, replacement := range replacements {
				newKey := strings.ReplaceAll(key, replacedKey, replacement)
				newKey = strings.ReplaceAll(newKey, "//", "/")
				addEntryIfNotPresent(newKey, value, originalDictionary, additionalEntries, prefixTree)
			}
		}
	}
}

func generateSZVariationForKey(key string, strokes []string, vowelDashRegex *regexp.Regexp, rightHandAfterS *regexp.Regexp,
	value string, originalDictionary *map[string]string, additionalEntries *map[string]string, prefixTree *PrefixTree) {
	if strings.HasSuffix(key, "/-S") || strings.HasSuffix(key, "/-Z") {
		previousStroke := strokes[len(strokes)-2]
		if vowelDashRegex.MatchString(previousStroke) {
			previousStroke = getPartAfterVowels(previousStroke)
		}
		if strings.HasSuffix(key, "/-S") && !strings.HasSuffix(previousStroke, "S") && !rightHandAfterS.MatchString(previousStroke) {
			keyVariation1 := strings.TrimSuffix(key, "/-S") + "Z"
			keyVariation2 := strings.TrimSuffix(key, "/-S") + "S"
			addEntryIfNotPresent(keyVariation1, value, originalDictionary, additionalEntries, prefixTree)
			addEntryIfNotPresent(keyVariation2, value, originalDictionary, additionalEntries, prefixTree)
		}
		if strings.HasSuffix(key, "/-Z") && !strings.HasSuffix(previousStroke, "Z") {
			keyVariation1 := strings.TrimSuffix(key, "/-Z") + "Z"
			keyVariation2 := strings.TrimSuffix(key, "/-Z") + "S"
			addEntryIfNotPresent(keyVariation1, value, originalDictionary, additionalEntries, prefixTree)
			addEntryIfNotPresent(keyVariation2, value, originalDictionary, additionalEntries, prefixTree)
		}
	}
}

func generateKwrRemovedVariations(key string, strokes []string, originalDictionary *map[string]string) [][]string {
	// Step 1: Find indexes where strokes[i] starts with "KWR" but is not equal to "KWR"
	indexes := []int{}
	for i, stroke := range strokes {
		if i > 0 && strings.HasPrefix(stroke, "KWR") && stroke != "KWR" {
			indexes = append(indexes, i)
		}
	}

	// Step 2: Generate all combinations of replacing KWR in strokes elements with ""
	replacementOptions := generateReplacementOptions(indexes)
	var variations [][]string
	for _, replacement := range replacementOptions {
		// Step 3: Apply the replacement options to a copy of strokes
		newStrokes := make([]string, len(strokes))
		copy(newStrokes, strokes)
		for i, shouldReplace := range replacement {
			if shouldReplace && indexes[i] > 0 {
				newStrokes[indexes[i]] = strings.TrimPrefix(newStrokes[indexes[i]], "KWR")
			}
		}

		// Step 4: Check if the result is distinct and valid
		if isDistinctAndValid(key, indexes, replacement, newStrokes, originalDictionary) {
			variations = append(variations, newStrokes)
		}
	}

	// sort variations
	slices.SortFunc(variations, func(a, b []string) int {
		return strings.Compare(strings.Join(a, "/"), strings.Join(b, "/"))
	})
	return variations
}

func generateReplacementOptions(indexes []int) [][]bool {
	options := [][]bool{}
	for i := 0; i < (1 << len(indexes)); i++ {
		replacement := make([]bool, len(indexes))
		for j := 0; j < len(indexes); j++ {
			replacement[j] = (i & (1 << j)) != 0
		}
		options = append(options, replacement)
	}
	return options
}

func isDistinctAndValid(key string, indexes []int, replacement []bool, strokes []string, originalDictionary *map[string]string) bool {
	if strings.Join(strokes, "/") == key {
		return false
	}

	for i, index := range indexes {
		if replacement[i] {
			joined := strings.Join(strokes[:index], "/")
			if hasKey(joined, originalDictionary) {
				return false
			}
		}
	}
	return true
}

func isConsonant(r rune) bool {
	consonants := "BCDFGHJKLMNPQRSTVWXZ"
	return strings.ContainsRune(consonants, r)
}

func generateIntervalCombinations(ranges [][]int) [][]int {
	result := [][]int{}
	current := make([]int, len(ranges))

	var generate func(int)
	generate = func(index int) {
		if index == len(ranges) {
			combination := make([]int, len(current))
			copy(combination, current)
			result = append(result, combination)
			return
		}

		start, end := ranges[index][0], ranges[index][1]
		for i := start; i <= end; i++ {
			current[index] = i
			generate(index + 1)
		}
	}

	generate(0)
	return result
}

func countConsonantsAtEnd(s string) int {
	count := 0
	for i := len(s) - 1; i >= 0; i-- {
		if !isConsonant(rune(s[i])) {
			break
		}
		count++
	}
	return count
}

func countConsonantsAtBeginning(s string) int {
	count := 0
	for i := 0; i < len(s); i++ {
		ch := string(s[i])
		chIsDash := ch == "-"
		if !isConsonant(rune(s[i])) && !chIsDash {
			break
		}
		count++
	}
	return count
}

func applyOffsetsToStrokes(strokes []string, offsets []int) [][]string {
	lhsStenoLetters := []string{
		"KWR",
		"PW",
		"KH",
		"TK",
		"TP",
		"TH",
		"TKPW",
		"EU",
		"SKWR",
		"HR",
		"PH",
		"TPH",
		"KW",
		"SR",
		"KP",
		"KWR",
		"STKPW",
		"SH",
		"KH",
		"THR",
	}
	rhsStenoLetters := []string{
		"FT",
		"PL",
		"BG",
		"BGT",
		"PBGT",
		"LG",
		"PB",
		"PBLG",
		"FRB",
		"PBG",
		"FP",
		"RB",
		"FRPB",
		"GS",
		"BGS",
		"PBT",
		"PLT",
		"LT",
		"BL",
		"PBS",
	}
	result := [][]string{}

	var generate func(int, []string)
	generate = func(index int, current []string) {
		if index == len(offsets) {
			combination := make([]string, len(current))
			copy(combination, current)
			result = append(result, combination)
			return
		}

		// Don't apply offset
		generate(index+1, current)

		// Apply offset
		if index < len(current)-1 {
			// Check if the second element starts with KWR followed by a vowel or PW
			shouldProcess := !isGlider(current[index+1])
			offset := offsets[index]
			if shouldProcess && offset < 0 {

				for _, letter := range rhsStenoLetters {
					lhsWord := current[index]
					movementAmountTooSmall := abs(offset) < len(letter)
					if movementAmountTooSmall && strings.HasSuffix(lhsWord, letter) {
						shouldProcess = false
						break
					}
				}
			}
			if shouldProcess {
				prefixLettersBeingMoved := offset > 0
				rhsWord := current[index+1]
				for _, letter := range lhsStenoLetters {
					movementAmountTooSmall := abs(offset) < len(letter) && strings.HasPrefix(rhsWord, letter)
					if prefixLettersBeingMoved && movementAmountTooSmall {
						shouldProcess = false
						break
					}
				}
				if shouldProcess {
					// check for -<right hand expression>
					for _, letter := range rhsStenoLetters {
						dashLetter := "-" + letter
						movementAmountTooSmallDash := abs(offset) < len(dashLetter) && strings.HasPrefix(rhsWord, dashLetter)
						if prefixLettersBeingMoved && movementAmountTooSmallDash {
							shouldProcess = false
							break
						}
					}
				}
			}
			if shouldProcess {
				newStrokes := make([]string, len(current))
				copy(newStrokes, current)

				// don't move *T and similar LHS strings around
				lhsHasAsterisk := strings.Contains(newStrokes[index], "*")
				if offset < 0 && !lhsHasAsterisk {
					// Move characters from first string to second
					moveChars := min(-offset, len(newStrokes[index]))
					lhsSuffix := newStrokes[index][len(newStrokes[index])-moveChars:]
					newStrokes[index+1] = moveLhsSuffixToRhsStroke(newStrokes[index+1], lhsSuffix)
					newStrokes[index] = newStrokes[index][:len(newStrokes[index])-moveChars]
				} else if offset > 0 {
					// Move characters from second string to first
					moveChars := min(offset, len(newStrokes[index+1]))
					rhsChars := newStrokes[index+1][:moveChars]
					// if we are moving a string like "-PLT", remove the "-" so it can be a valid stroke
					if strings.HasPrefix(rhsChars, "-") && len(rhsChars) > 1 {
						rhsChars = strings.TrimPrefix(rhsChars, "-")
					}
					newStrokes[index] = moveRhsPrefixToLhsStroke(newStrokes[index], rhsChars)
					newStrokes[index+1] = newStrokes[index+1][moveChars:]
				}

				generate(index+1, newStrokes)
			}
		}
	}

	generate(0, strokes)

	if len(result) <= 1 {
		return result
	}

	// filter uniquevalues in `result`
	uniqueValues := make(map[string]bool)
	uniqueValues[strings.Join(strokes, "/")] = true
	var uniqueValuesList [][]string
	for _, combination := range result {
		uniqueValue := strings.Join(combination, "/")
		if _, ok := uniqueValues[uniqueValue]; !ok {
			uniqueValues[uniqueValue] = true
			uniqueValuesList = append(uniqueValuesList, combination)
		}
	}
	return uniqueValuesList
}

func returnIfContains(list string, char string) string {
	if strings.Contains(list, char) {
		return char
	} else {
		return ""
	}
}

func moveRhsPrefixToLhsStroke(lhs, rhsPrefix string) string {
	alteredRhsLetters := make(map[string]string)
	//           left hand    right hand
	alteredRhsLetters["PW"] = "B"      // B
	alteredRhsLetters["TK"] = "D"      // D
	alteredRhsLetters["TP"] = "F"      // F
	alteredRhsLetters["TKPW"] = "G"    // G
	alteredRhsLetters["SKWR"] = "PBLG" // J
	alteredRhsLetters["K"] = "BG"      // K
	alteredRhsLetters["HR"] = "L"      // L
	alteredRhsLetters["PH"] = "PL"     // M
	alteredRhsLetters["TPH"] = "PB"    // N
	alteredRhsLetters["SR"] = "F"      // V
	alteredRhsLetters["TH"] = "*T"     // TH
	alteredRhsLetters["KH"] = "FP"
	alteredRhsLetters["SH"] = "RB"
	alteredRhsLetters["SR"] = "F"    // V
	alteredRhsLetters["STKPW"] = "Z" // Z
	if _, ok := alteredRhsLetters[rhsPrefix]; ok {
		lookup := alteredRhsLetters[rhsPrefix]
		if strings.HasPrefix(lookup, "*") {
			lhsParts := separateStrokeParts(lhs)
			lhsVowels := returnIfContains(lhsParts.Vowels, "A") +
				returnIfContains(lhsParts.Vowels, "O") +
				"*" +
				returnIfContains(lhsParts.Vowels, "E") +
				returnIfContains(lhsParts.Vowels, "U")
			return lhsParts.Left + lhsVowels + lhsParts.Right + lookup[1:]
		} else {
			return lhs + lookup
		}
	}
	return lhs + rhsPrefix
}

func moveLhsSuffixToRhsStroke(rhs, lhsPrefix string) string {
	alteredLhsLetters := make(map[string]string)
	//           right hand    left hand
	alteredLhsLetters["PL"] = "PH"     // M
	alteredLhsLetters["TPH"] = "PB"    // N
	alteredLhsLetters["F"] = "TP"      // V
	alteredLhsLetters["BG"] = "K"      // K
	alteredLhsLetters["BGT"] = "-BGT"  // KT
	alteredLhsLetters["PBLG"] = "SKWR" // J
	alteredLhsLetters["FP"] = "CH"
	alteredLhsLetters["RB"] = "SH"
	rhsWithoutDash := strings.TrimPrefix(rhs, "-")
	if _, ok := alteredLhsLetters[lhsPrefix]; ok {
		lookup := alteredLhsLetters[lhsPrefix]
		return lookup + rhsWithoutDash
	}

	return lhsPrefix + rhsWithoutDash
}
func abs(index int) int {
	if index < 0 {
		return -index
	} else {
		return index
	}
}

func isGlider(stroke string) bool {
	if len(stroke) < 4 {
		return false
	}
	if stroke[:3] != "KWR" {
		return false
	}
	return isVowel(stroke[3])
}

func isVowel(r byte) bool {
	vowels := "AEOU"
	return strings.ContainsRune(vowels, rune(r))
}

func generateAlternateSyllableSplitStrokes(strokes []string, originalDictionary *map[string]string, additionalEntries *map[string]string, prefixTree *PrefixTree) [][]string {
	var intervals [][]int

	for i := 0; i <= len(strokes)-2; i++ {
		firstStroke := strokes[i]
		secondStroke := strokes[i+1]
		intervalLeft := -countConsonantsAtEnd(firstStroke)
		intervalRight := countConsonantsAtBeginning(secondStroke)
		intervals = append(intervals, []int{intervalLeft, intervalRight})
	}
	intervalCombinations := generateIntervalCombinations(intervals)

	var alternateStrokes [][]string

	uniqueStrokes := make(map[string]bool)
	originalStrokes := strings.Join(strokes, "/")
	uniqueStrokes[originalStrokes] = true

	for _, combination := range intervalCombinations {
		appliedStrokes := applyOffsetsToStrokes(strokes, combination)
		for _, strokeSet := range appliedStrokes {
			validStrokes := true
			for _, stroke := range strokeSet {
				if !isValidStenoOrder(stroke) {
					validStrokes = false
					break
				}
			}
			if !validStrokes {
				continue
			}
			validStrokes = validWordBoundaries(strokeSet, originalDictionary, additionalEntries, prefixTree)
			if validStrokes {
				// filter elements of strokeSet that are empty
				strokeSet = removeEmpty(strokeSet)
				joinedStrokes := strings.Join(strokeSet, "/")
				if !uniqueStrokes[joinedStrokes] {
					alternateStrokes = append(alternateStrokes, strokeSet)
					uniqueStrokes[joinedStrokes] = true
				}
			}
		}
	}

	// sort alternateStrokes
	slices.SortFunc(alternateStrokes, func(a, b []string) int {
		return strings.Compare(strings.Join(a, "/"), strings.Join(b, "/"))
	})
	return alternateStrokes
}

func validWordBoundaries(strokeSet []string, originalDictionary *map[string]string, additionalEntries *map[string]string, prefixTree *PrefixTree) bool {
	if len(strokeSet) < 2 {
		return true
	}

	// check from right to left
	for strokesBack := 1; strokesBack < len(strokeSet); strokesBack++ {
		splitPoint := len(strokeSet) - strokesBack
		suffixStrokes := strokeSet[splitPoint:]
		suffix := strings.Join(suffixStrokes, "/")
		prefixStrokes := strokeSet[:splitPoint]
		prefix := strings.Join(prefixStrokes, "/")
		if (hasKey(prefix, additionalEntries) || hasKey(prefix, originalDictionary)) &&
			(hasKey(suffix, additionalEntries) || hasKey(suffix, originalDictionary) || prefixTree.HasPrefix(suffixStrokes)) {
			prefixValue := (*originalDictionary)[prefix]
			if !strings.HasSuffix(prefixValue, "^}") {
				suffixValue := (*originalDictionary)[suffix]
				if !strings.HasPrefix(suffixValue, "{^") {
					return false
				}
			}
		}
	}
	// now check from left to right
	for strokesForward := 1; strokesForward < len(strokeSet); strokesForward++ {
		splitPoint := strokesForward
		prefixStrokes := strokeSet[:splitPoint]
		prefix := strings.Join(prefixStrokes, "/")
		suffixStrokes := strokeSet[splitPoint:]
		suffix := strings.Join(suffixStrokes, "/")
		if (hasKey(prefix, additionalEntries) || hasKey(prefix, originalDictionary)) &&
			(hasKey(suffix, additionalEntries) || hasKey(suffix, originalDictionary) || prefixTree.HasPrefix(suffixStrokes)) {
			prefixValue := (*originalDictionary)[prefix]
			if !strings.HasSuffix(prefixValue, "^}") {
				suffixValue := (*originalDictionary)[suffix]
				if !strings.HasPrefix(suffixValue, "{^") {
					return false
				}
			}
		}
	}
	// another form of possible outline conflict we might want to avoid is like when we have:
	// <stroke 1>/<stroke 2>/<stroke 3>/<stroke 4>
	// where stroke 1 already exists and so do strokes 2 and 3
	// if len(strokeSet) >= 3 {
	// 	for validPrefixStrokes := 1; validPrefixStrokes < len(strokeSet)-1; validPrefixStrokes++ {
	// 		prefixStrokes := strokeSet[:validPrefixStrokes]
	// 		prefix := strings.Join(prefixStrokes, "/")
	// 		if hasKey(prefix, originalDictionary) || hasKey(prefix, additionalEntries) {
	// 			for suffixPoint := validPrefixStrokes + 1; suffixPoint < len(strokeSet); suffixPoint++ {
	// 				suffix := strings.Join(strokeSet[validPrefixStrokes:suffixPoint], "/")
	// 				if hasKey(suffix, originalDictionary) || hasKey(suffix, additionalEntries) {
	// 					return false
	// 				}
	// 			}
	// 		} else {
	// 			break
	// 		}
	// 	}
	// }
	return true
}

func removeEmpty(strokeSet []string) []string {
	for i := len(strokeSet) - 1; i >= 0; i-- {
		if strokeSet[i] == "" {
			strokeSet = append(strokeSet[:i], strokeSet[i+1:]...)
		}
	}
	return strokeSet
}

func getPartAfterVowels(input string) string {
	vowels := "AEOU-*"
	lastVowelIndex := -1
	for i := range input {
		if strings.ContainsRune(vowels, rune(input[i])) {
			lastVowelIndex = i
		}
	}
	if lastVowelIndex != -1 {
		return input[lastVowelIndex+1:]
	}
	return input
}

func hasKey(key string, dict *map[string]string) bool {
	_, ok := (*dict)[key]
	return ok
}

func addEntryIfNotPresent(key, value string, originalDict *map[string]string, additionalDict *map[string]string, prefixTree *PrefixTree) bool {
	if !hasKey(key, originalDict) && !hasKey(key, additionalDict) {
		strokes := strings.Split(key, "/")
		if !validWordBoundaries(strokes, originalDict, additionalDict, prefixTree) { // check if there is a conflict
			return false
		}
		for _, stroke := range strokes {
			if !isValidStenoOrder(stroke) {
				return false
			}
		}
		(*additionalDict)[key] = value
		return true
	}
	return false
}

type StenoParts struct {
	Left   string
	Vowels string
	Right  string
	Valid  bool
}

func separateStrokeParts(stroke string) StenoParts {
	left := "ZSTKPWHRV"
	vowels := "AO*EU"
	right := "FRPBLGTSDZ"

	var parts StenoParts
	parts.Valid = true
	state := 0 // 0: left, 1: vowels, 2: right

	for _, ch := range stroke {
		c := string(ch)
		switch state {
		case 0: // Left
			if strings.ContainsAny(c, left) {
				parts.Left += c
			} else if strings.ContainsAny(c, vowels) {
				state = 1
				parts.Vowels += c
			} else if strings.ContainsAny(c, right) {
				state = 2
				parts.Right += c
			} else {
				parts.Valid = false
				return parts
			}
		case 1: // Vowels
			if strings.ContainsAny(c, vowels) {
				parts.Vowels += c
			} else if strings.ContainsAny(c, right) {
				state = 2
				parts.Right += c
			} else {
				parts.Valid = false
				return parts
			}
		case 2: // Right
			if strings.ContainsAny(c, right) {
				parts.Right += c
			} else {
				parts.Valid = false
				return parts
			}
		}
	}

	return parts
}

func hasConsecutiveRepeatedLetters(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] == s[i-1] {
			return true
		}
	}
	return false
}

func isValidStenoOrder(stroke string) bool {
	if hasConsecutiveRepeatedLetters(stroke) || stroke == "-" {
		return false
	}

	if strings.HasPrefix(stroke, "-") {
		return isValidOrder(stroke[1:], "FRPBLGTSDZ")
	}

	parts := separateStrokeParts(stroke)
	if !parts.Valid {
		return false
	}

	return isValidOrder(parts.Left, "ZSTKPWHRV") &&
		isValidOrder(parts.Vowels, "AO*EU") &&
		isValidOrder(parts.Right, "FRPBLGTSDZ")
}

func isValidOrder(substr, order string) bool {
	lastIndex := -1
	for _, ch := range substr {
		index := strings.IndexRune(order, ch)
		if index == -1 || index < lastIndex {
			return false
		}
		lastIndex = index
	}
	return true
}
