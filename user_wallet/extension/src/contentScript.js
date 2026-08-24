// Content script: injects the inpage provider into the MAIN world and relays
// JSON-RPC between the page and the extension background (ISOLATED world).
(function () {
  'use strict';

  // Inject the provider as a <script> so it runs in the page's main world.
  const script = document.createElement('script');
  script.src = chrome.runtime.getURL('src/inpage.js');
  script.onload = () => script.remove();
  (document.head || document.documentElement).appendChild(script);

  // Relay inpage requests -> background, and background replies -> inpage.
  window.addEventListener('message', (ev) => {
    if (ev.source !== window) return;
    const msg = ev.data;
    if (!msg || msg.__tigerwallet !== 'inpage-request') return;
    chrome.runtime.sendMessage(
      { kind: 'provider-rpc', id: msg.id, method: msg.method, params: msg.params, origin: location.origin },
      (response) => {
        if (chrome.runtime.lastError) {
          window.postMessage({
            __tigerwallet: 'inpage-response', id: msg.id,
            error: { code: 4900, message: 'UserWallet extension unavailable' },
          }, '*');
          return;
        }
        window.postMessage({ __tigerwallet: 'inpage-response', id: msg.id, ...response }, '*');
      }
    );
  });

  // Forward extension-originated events (account/chain changes) to the page.
  chrome.runtime.onMessage.addListener((msg) => {
    if (msg && msg.kind === 'provider-event') {
      window.postMessage({ __tigerwallet: 'inpage-response', type: 'event', event: msg.event, payload: msg.payload }, '*');
    }
  });
})();
