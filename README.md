# lapwing_augmentor
Augment the <a href="https://raw.githubusercontent.com/aerickt/plover-lapwing-aio/main/plover_lapwing/dictionaries/lapwing-base.json">Lapwing dictionary</a> with alternatives for vowels and a few other cases.

Warning: please note that this is an initial experiment and should only be used with caution if you do not mind having potentially invalid strokes in your dictionary. It will also generate dozens of questionable/debatable outlines for words like `possibilities`. Don't run this if you don't like the idea of that happening.

This has not been thoroughly checked yet.

Current additions:

- for outlines ending with `/KWREU`, add `/KWRAE` and `/KWRAOE` variants. Rationale: `/KWREU` seemed odd and I would have expected it to be `/KWRAOE`. Normally this doesn't conflict except in cases like `trustee` vs. `trusty`. Phoenix theory apparently uses `AE` so I wanted to add that as an option.
- fold some `/-<letter>/KWREU` outlines into a single stroke: `/-B/KWREU` -> `/PWEU`, `/PWAE`, `/PWAOE`. Rationale: this safely reduces strokes and seems intuitive. Also, `R/KWREU` gets replaced with `/REU`, `/RAE`, and `/RAOE`. The rationale is that this seems more consistent with the Lapwing splitting rules of having a consonant at the beginning of the stroke, effectively ignoring cases where r is treated by the base dictionary as a vowel. This effectively nullifies https://lapwing.aerick.ca/Chapter-15.html#kwr-with-the--r-key .
- initial experimentation in generating alternate splits, e.g. finding other valid ways to split words like "distribute". this code adds `"TKEU/STREU/PWAOUT"` to compliment Lapwing's `"TKEUS/TREU/PWAOUT`. this is still in progress and there are probably a lot of invalid strokes.
- remove KWR in outlines where it should be safe and not create word boundary ambiguity
- try to fold in `-<letter(s)>` strokes into the previous stroke if it is legal steno
- all additions above are only added if it doesn't create a word outline conflict

If we're going to deviate from Lapwing's rules, why even use Lapwing? Well, this way we still get most of the benefits of Lapwing, we are just tweaking the parts that we disagree with. Lapwing's syllabic splitting rules are still more consisten and sensible. This adds a few variations that make Lapwing a little more flexible, but overall keep a fairly logical structure.

Usage: 

```
$ lapwing_augmentor --lapwing_source <source-dict> [--lapwing_source <source-dict2> ...] --output_target <target-dict> [--output_target <target-dict2> ...] 
```

The code will read in every pat passed with the `--lapwing_source` parameter, so you can give it multiple paths if e.g. you want it to process both Lapwing and your own personal dictionaries:

```
./lapwing_augmentor --lapwing_source ../aerick-steno-dictionaries/lapwing-base.json --lapwing_source ../steno-dictionaries/lapwing-additions.json --output_target ../steno-dictionaries/lapwing-augmentations.json
```

You can have a look at <a href="lapwing-augmentations-current-output.json">the current output</a> that results from only running against `lapwing-base.json`.