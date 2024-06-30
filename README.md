# lapwing_augmentor
Augment the <a href="https://raw.githubusercontent.com/aerickt/plover-lapwing-aio/main/plover_lapwing/dictionaries/lapwing-base.json">Lapwing dictionary</a> with alternatives for vowels and a few other cases.

Warning: please note that this is an initial experiment and should only be used with caution if you do not mind having potentially invalid strokes in your dictionary.

This has not been thoroughly checked yet.

Current additions:

- for outlines ending with `/KWREU`, add `/KWRAE` and `/KWRAOE` variants. Rationale: `/KWREU` seemed odd and I would have expected it to be `/KWRAOE`. Normally this doesn't conflict except in cases like `trustee` vs. `trusty`. Phoenix theory apparently uses `AE` so I wanted to add that as an option.
- fold some `/-<letter>/KWREU` outlines into a single stroke: `/-B/KWREU` -> `/PWEU`, `/PWAE`, `/PWAOE`. Rationale: this safely reduces strokes and seems intuitive.
- for outlines ending with `/-S` or `/-Z` that can safely add on the S/Z to the end, make a new outline with the S and Z added. Rationale: this reduces strokes and I think it is fine to tack it on to the end if you want to do that.
- all additions above are only added if it doesn't create a word outline conflict

Usage: 

```
$ lapwing_augmentor --lapwing_source <source-dict> --output_target <target-dict>
```

To see the current output, have a look at <a href="lapwing-augmentations-current-output.json">the current output</a>.