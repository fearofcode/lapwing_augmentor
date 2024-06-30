# lapwing_augmentor
Augment the Lapwing dictionary with alternatives for vowels and a few other cases.

Please note that this is an initial experiment and should only be used with caution if you do not mind having potentially invalid strokes in your dictionary.

This has not been thoroughly checked yet.

Current additions:

- for outlines ending with `/KWREU`, add `/KWRAE` and `/KWRAOE` variants
- fold some `/-<letter>/KWREU` outlines into a single stroke: `/-B/KWREU` -> `/PWEU`, `/PWAE`, `/PWAOE`
- for outlines ending with `/-S` or `/-Z` that can safely add on the S/Z to the end, make a new outline with the S and Z added
- all additions above are only added if it doesn't create a word outline conflict

Usage: 

```
$ lapwing_augmentor --lapwing_source <source-dict> --output_target <target-dict>
```

To see the current output, have a look at <a href="lapwing-augmentations-current-output.json">the current output</a>.