/**
 * TigerWallet - Content Script
 * Injects provider into pages
 */

// Inject the provider
function injectProvider() {
  const script = document.createElement('script');
  script.src = chrome.runtime.getURL('injected/injected.js');
  script.onload = function() {
    script.remove();
  };
  (document.head || document.documentElement).appendChild(script);
}

// Listen for page load
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', injectProvider);
} else {
  injectProvider();
}

// Also inject on frame loads
window.addEventListener('load', injectProvider);
