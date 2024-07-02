package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
)

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
		sourceDictPath  string
		targetDictPaths stringList
	)
	flag.StringVar(&sourceDictPath, "lapwing_source", "", "source dictionary path")
	flag.Var(&targetDictPaths, "output_target", "target dictionary path(s)")
	flag.Parse()

	if sourceDictPath == "" || len(targetDictPaths) == 0 {
		fmt.Println("Usage: lapwing_augmentor --lapwing_source <source-dict> --output_target <target-dict> [--output_target <target-dict2> ...]")
		os.Exit(1)
	}

	logger.Println("Reading in dictionary from ", sourceDictPath)

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

	originalDictionary := make(map[string]string)
	for key, value := range sourceJSON {
		originalDictionary[key] = value.(string)
	}
	logger.Println("Done reading in dictionary. Size:", len(originalDictionary))

	additionalEntries := make(map[string]string)
	kwrSuffixEndPattern := `^.*/KWR([^/]+)$`
	kwrSuffixEndRegex := regexp.MustCompile(kwrSuffixEndPattern)
	suffixReplacements := make(map[string][]string)
	suffixReplacements["/-B/KWREU"] = []string{"/PWEU", "/PWAOE", "/PWAE"}
	suffixReplacements["/-BL/KWREU"] = []string{"/PWHREU", "/PWHRAOE", "/PWHRAE"}
	suffixReplacements["/-FL/KWREU"] = []string{"/TPHREU", "/TPHRAOE", "/TPHRAE"}
	suffixReplacements["/-L/KWREU"] = []string{"/HREU", "/HRAOE", "/HRAE"}
	suffixReplacements["/-P/KWREU"] = []string{"/PEU", "/PAOE", "/PAE"}
	suffixReplacements["/-PL/KWREU"] = []string{"/PHREU", "/PHRAOE", "/PHRAE"}
	suffixReplacements["R/KWREU"] = []string{"/REU", "/RAOE", "/RAE"}
	vowelsDashes := `[AEOU\-*]+`
	vowelDashRegex := regexp.MustCompile(vowelsDashes)
	rightHandAfterS := regexp.MustCompile(`[DZ]`)
	for key, value := range originalDictionary {
		strokes := strings.Split(key, "/")
		if len(strokes) >= 2 {
			alternateStrokes := generateAlternateSyllableSplitStrokes(strokes, &originalDictionary, &additionalEntries)
			for _, strokeSet := range alternateStrokes {
				addEntryIfNotPresent(key, strings.Join(strokeSet, "/"), value, &originalDictionary, &additionalEntries, logger)
			}

			// look for cases where we can safely remove KWR without creating word boundary errors
			if strings.Contains(key, "/KWR") {
				variations := generateKwrRemovedVariations(key, strokes, &originalDictionary)
				for _, variation := range variations {
					addEntryIfNotPresent(key, strings.Join(variation, "/"), value, &originalDictionary, &additionalEntries, logger)
				}
			}
		}

		// check if we can safely fold in a /-D to the end
		if strings.HasSuffix(key, "/-D") {
			previousStroke := strokes[len(strokes)-2]
			if vowelDashRegex.MatchString(previousStroke) {
				previousStroke = getPartAfterVowels(previousStroke)
			}
			if !strings.HasSuffix(previousStroke, "D") {
				keyVariation := strings.TrimSuffix(key, "/-D") + "D"
				addEntryIfNotPresent(key, keyVariation, value, &originalDictionary, &additionalEntries, logger)
				// now see if we can also fold in S/Z
				keyStrokes := strings.Split(keyVariation, "/")
				generateSZVariationForKey(keyVariation, keyStrokes, vowelDashRegex, rightHandAfterS, value, originalDictionary, additionalEntries, logger)
			}
		}
		if strings.HasSuffix(key, "/-G") {
			previousStroke := strokes[len(strokes)-2]
			if vowelDashRegex.MatchString(previousStroke) {
				previousStroke = getPartAfterVowels(previousStroke)
			}
			if !strings.HasSuffix(previousStroke, "G") {
				keyVariation := strings.TrimSuffix(key, "/-G") + "G"
				addEntryIfNotPresent(key, keyVariation, value, &originalDictionary, &additionalEntries, logger)
				// now see if we can also fold in S/Z
				keyStrokes := strings.Split(keyVariation, "/")
				generateSZVariationForKey(keyVariation, keyStrokes, vowelDashRegex, rightHandAfterS, value, originalDictionary, additionalEntries, logger)
			}
		}

		generateSZVariationForKey(key, strokes, vowelDashRegex, rightHandAfterS, value, originalDictionary, additionalEntries, logger)

		for replacedSuffix, replacements := range suffixReplacements {
			if strings.HasSuffix(key, replacedSuffix) {
				for _, replacement := range replacements {
					newKey := strings.TrimSuffix(key, replacedSuffix) + replacement
					addEntryIfNotPresent(key, newKey, value, &originalDictionary, &additionalEntries, logger)
				}
				break
			}
		}

		kwrMatch := kwrSuffixEndRegex.FindStringSubmatch(key)
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
				addEntryIfNotPresent(key, keyVariation1, value, &originalDictionary, &additionalEntries, logger)
				keyVariation2 := fmt.Sprintf("%sAE", kwrPrefix)
				addEntryIfNotPresent(key, keyVariation2, value, &originalDictionary, &additionalEntries, logger)
			}
		}

	}

	// see if we can generate KWR removed variations on additional entries we just generated
	for key, value := range additionalEntries {
		strokes := strings.Split(key, "/")
		if len(strokes) >= 2 && strings.Contains(key, "/KWR") {
			variations := generateKwrRemovedVariations(key, strokes, &originalDictionary)
			for _, variation := range variations {
				addEntryIfNotPresent(key, strings.Join(variation, "/"), value, &originalDictionary, &additionalEntries, logger)
			}
		}
	}

	log.Println("Added", len(additionalEntries), "additional entries")

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

func generateSZVariationForKey(key string, strokes []string, vowelDashRegex *regexp.Regexp, rightHandAfterS *regexp.Regexp, value string, originalDictionary map[string]string, additionalEntries map[string]string, logger *log.Logger) {
	if strings.HasSuffix(key, "/-S") || strings.HasSuffix(key, "/-Z") {
		previousStroke := strokes[len(strokes)-2]
		if vowelDashRegex.MatchString(previousStroke) {
			previousStroke = getPartAfterVowels(previousStroke)
		}
		if strings.HasSuffix(key, "/-S") && !strings.HasSuffix(previousStroke, "S") && !rightHandAfterS.MatchString(previousStroke) {
			keyVariation1 := strings.TrimSuffix(key, "/-S") + "Z"
			keyVariation2 := strings.TrimSuffix(key, "/-S") + "S"
			addEntryIfNotPresent(key, keyVariation1, value, &originalDictionary, &additionalEntries, logger)
			addEntryIfNotPresent(key, keyVariation2, value, &originalDictionary, &additionalEntries, logger)
		}
		if strings.HasSuffix(key, "/-Z") && !strings.HasSuffix(previousStroke, "Z") {
			keyVariation1 := strings.TrimSuffix(key, "/-Z") + "Z"
			keyVariation2 := strings.TrimSuffix(key, "/-Z") + "S"
			addEntryIfNotPresent(key, keyVariation1, value, &originalDictionary, &additionalEntries, logger)
			addEntryIfNotPresent(key, keyVariation2, value, &originalDictionary, &additionalEntries, logger)
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
		if !isConsonant(rune(s[i])) {
			break
		}
		count++
	}
	return count
}

func applyOffsetsToStrokes(strokes []string, offsets []int) [][]string {
	stenoLetters := []string{
		"PW",
		"KH",
		"TK",
		"TP",
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
		// todo we only need to check if these are on RHS
		"FT",
		"PL",
		"BG",
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
		"PLT",
		"LT",
		"BL",
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
			for _, letter := range stenoLetters {
				if !wontDisruptLettersOnEitherStroke(offset, current[index], current[index+1], letter) {
					shouldProcess = false
					break
				}
			}
			if shouldProcess {
				newStrokes := make([]string, len(current))
				copy(newStrokes, current)

				if offset < 0 {
					// Move characters from first string to second
					moveChars := min(-offset, len(newStrokes[index]))
					newStrokes[index+1] = newStrokes[index][len(newStrokes[index])-moveChars:] + newStrokes[index+1]
					newStrokes[index] = newStrokes[index][:len(newStrokes[index])-moveChars]
				} else {
					// Move characters from second string to first
					moveChars := min(offset, len(newStrokes[index+1]))
					newStrokes[index] = newStrokes[index] + newStrokes[index+1][:moveChars]
					newStrokes[index+1] = newStrokes[index+1][moveChars:]
				}

				generate(index+1, newStrokes)
			}
		}
	}

	generate(0, strokes)
	return result
}

func wontDisruptLettersOnEitherStroke(offset int, lhs, rhs, prefix string) bool {
	return abs(offset) >= len(prefix) || (!strings.HasSuffix(lhs, prefix) && !strings.HasPrefix(rhs, prefix))
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

func generateAlternateSyllableSplitStrokes(strokes []string, originalDictionary *map[string]string, additionalEntries *map[string]string) [][]string {
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
			// TODO do we need to check this for last 2 and rest of strokes, etc?
			lastStroke := strokeSet[len(strokeSet)-1]
			// get all but last element of strokeSet
			suffixStrokes := strokeSet[:len(strokeSet)-1]
			suffix := strings.Join(suffixStrokes, "/")
			if (hasKey(suffix, originalDictionary) && hasKey(lastStroke, originalDictionary)) ||
				(hasKey(suffix, additionalEntries) && hasKey(lastStroke, additionalEntries)) {
				validStrokes = false
			}
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
	return alternateStrokes
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

func addEntryIfNotPresent(originalKey, key, value string, originalDict *map[string]string, additionalDict *map[string]string, logger *log.Logger) bool {
	if !hasKey(key, originalDict) && !hasKey(key, additionalDict) {
		strokes := strings.Split(key, "/")
		for _, stroke := range strokes {
			if !isValidStenoOrder(stroke) {
				return false
			}
		}
		(*additionalDict)[key] = value
		return true
	} else {
		logger.Println("Already has key:", key, "value:", value, "for original key:", originalKey)
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
	left := "STKPWHR"
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

	return isValidOrder(parts.Left, "STKPWHR") &&
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
