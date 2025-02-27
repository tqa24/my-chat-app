// frontend/vue.config.js
const { defineConfig } = require('@vue/cli-service')
module.exports = defineConfig({
  transpileDependencies: true,
  publicPath: '/static/', //  Tell Vue CLI to use /static/ as the base
})