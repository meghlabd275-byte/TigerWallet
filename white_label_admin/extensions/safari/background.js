// WebExtension API compatibility shim.
// Firefox MV2 exposes the promise-based `browser.*` namespace but not
// `chrome.*` promises; Chromium MV3 exposes `chrome.*` (with promises in
// service workers) but not `browser.*`. Alias whichever is present so the
// same source runs in both engines without per-browser code copies.
if (typeof browser !== "undefined" && typeof chrome === "undefined") {
  // eslint-disable-next-line no-global-assign
  chrome = browser;
} else if (typeof chrome !== "undefined" && typeof browser === "undefined") {
  // eslint-disable-next-line no-global-assign
  browser = chrome;
}
// Safari background script
safari.extension.installContentScript = function() {
  console.log('TigerWallet White Label Admin Safari Extension installed');
};
