import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { applyTheme } from './theme'
import { applyPalette } from './palette'
import { initTelegram } from './telegram'
import { captureAttribution } from './attribution'

applyTheme()
applyPalette()
initTelegram()
captureAttribution()

createApp(App).use(createPinia()).use(router).mount('#app')
