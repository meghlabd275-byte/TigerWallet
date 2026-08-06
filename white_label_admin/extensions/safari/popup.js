// Popup script
document.addEventListener('DOMContentLoaded', () => {
  document.getElementById('open-dashboard').addEventListener('click', () => {
    safari.application.activeBrowserWindow.openTab().url = 'http://localhost:3001';
  });
});
