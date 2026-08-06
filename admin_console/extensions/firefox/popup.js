// Popup script
document.addEventListener('DOMContentLoaded', () => {
  document.getElementById('open-dashboard').addEventListener('click', () => {
    browser.tabs.create({ url: 'http://localhost:3002' });
  });
});
