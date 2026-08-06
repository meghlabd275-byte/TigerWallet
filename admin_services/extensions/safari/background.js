// Safari background script
safari.extension.installContentScript = function() {
  console.log('TigerWallet Admin Services Safari Extension installed');
};

safari.extension.settings = {
  get: function(key, callback) {
    if (callback) callback(false);
    return false;
  },
  set: function(key, value) {}
};
