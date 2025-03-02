const { defineConfig } = require('@vue/cli-service');

module.exports = defineConfig({
  transpileDependencies: true,
  publicPath: '/static/', // Tell Vue CLI to use /static/ as the base
  productionSourceMap: false, // Disable source maps in production

  chainWebpack: config => {
    config.plugin('html').tap(args => {
      args[0].title = 'Chat App';
      return args;
    });
  }
});