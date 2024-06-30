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

func main() {
	logger := log.New(os.Stdout, "LOG: ", log.LstdFlags|log.Lmicroseconds)
	var (
		sourceDictPath string
		targetDictPath string
	)
	flag.StringVar(&sourceDictPath, "lapwing_source", "", "source dictionary path")
	flag.StringVar(&targetDictPath, "output_target", "", "target dictionary path")
	flag.Parse()

	if sourceDictPath == "" || targetDictPath == "" {
		fmt.Println("Usage: lapwing_augmentor --lapwing_source <source-dict> --output_target <target-dict>")
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
	vowelDashRegex := regexp.MustCompile(`[AEIOU-]`)
	rightHandAfterS := regexp.MustCompile(`[DZ]`)
	for key, value := range originalDictionary {
		if strings.HasSuffix(key, "/-S") || strings.HasSuffix(key, "/-Z") {
			// check if we can safely add -S or -Z to the previous stroke
			strokes := strings.Split(key, "/")
			stroke := strokes[len(strokes)-2]
			if vowelDashRegex.MatchString(stroke) {
				// get part of stroke after vowelRegex match
				afterVowel := getPartAfterVowels(stroke)
				if strings.HasSuffix(key, "/-S") && !strings.HasSuffix(afterVowel, "S") && !rightHandAfterS.MatchString(afterVowel) {
					keyVariation1 := strings.TrimSuffix(key, "/-S") + "S"
					addEntryIfNotPresent(key, keyVariation1, value, &originalDictionary, &additionalEntries, logger)
					keyVariation2 := strings.TrimSuffix(key, "/-S") + "Z"
					addEntryIfNotPresent(key, keyVariation2, value, &originalDictionary, &additionalEntries, logger)
				}
				if strings.HasSuffix(key, "/-Z") && !strings.HasSuffix(afterVowel, "Z") {
					keyVariation2 := strings.TrimSuffix(key, "/-Z") + "Z"
					addEntryIfNotPresent(key, keyVariation2, value, &originalDictionary, &additionalEntries, logger)
				}
			} else {
				if strings.HasSuffix(key, "/-S") && !strings.HasSuffix(stroke, "S") && !rightHandAfterS.MatchString(stroke) {
					keyVariation1 := strings.TrimSuffix(key, "/-S") + "S"
					addEntryIfNotPresent(key, keyVariation1, value, &originalDictionary, &additionalEntries, logger)

					keyVariation2 := strings.TrimSuffix(key, "/-S") + "Z"
					addEntryIfNotPresent(key, keyVariation2, value, &originalDictionary, &additionalEntries, logger)
				}
				if strings.HasSuffix(key, "/-Z") && !strings.HasSuffix(stroke, "Z") {
					keyVariation2 := strings.TrimSuffix(key, "/-Z") + "Z"
					addEntryIfNotPresent(key, keyVariation2, value, &originalDictionary, &additionalEntries, logger)
				}
			}
		}

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

	log.Println("Added", len(additionalEntries), "additional entries")

	// write out additionalEntries to file at targetDictPath
	contents, err := json.MarshalIndent(additionalEntries, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(targetDictPath, contents, 0644); err != nil {
		fmt.Println("Error writing to target dictionary:", err)
		os.Exit(1)
	}

	log.Println("Wrote", len(additionalEntries), "additional entries to", targetDictPath)
}

func getPartAfterVowels(input string) string {
	vowels := "aeiouAEIOU"
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

func addEntryIfNotPresent(originalKey, key, value string, originalDict *map[string]string, additionalDict *map[string]string, logger *log.Logger) {
	if !hasKey(key, originalDict) && !hasKey(key, additionalDict) {
		(*additionalDict)[key] = value
	} else {
		logger.Println("Already has key:", key, "value:", value, "for original key:", originalKey)
	}
}
