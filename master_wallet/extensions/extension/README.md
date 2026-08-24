# MasterWallet Browser Extension (single canonical source)

One extension source supports all five browsers. Previously this directory
held five byte-identical copies (`chrome/`, `brave_extension/`,
`edge_extension/`, `firefox_extension/`, `safari_extension/`) that drifted
and were impossible to keep in sync; they are now consolidated here.

## Layout

- `manifest.json` — canonical Chrome (MV3) manifest
- `manifests/manifest.<browser>.json` — per-browser manifest variants:
  - `brave`, `edge`: Chromium MV3 (description differs only)
  - `firefox`: MV2 with `background.scripts` and a gecko application id
  - `safari`: MV3 web extension (convert with `xcrun safari-web-extension-converter`)
- `background.js`, `injected.js`, `popup.html`, `popup.js`, `services/` —
  shared sources, byte-identical across all browsers

## Build

```bash
./build.sh chrome   # -> dist/chrome
./build.sh firefox  # -> dist/firefox
./build.sh safari   # -> dist/safari (then run the Xcode converter)
```

The build copies the shared sources plus the browser-specific manifest into
`dist/<browser>/`, which can be loaded unpacked (Chrome/Brave/Edge/Firefox)
or converted for Safari.
