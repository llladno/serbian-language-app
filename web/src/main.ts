import { createApp } from 'vue'
import { createPinia } from 'pinia'
import './style.css'
import App from './App.vue'
import router from './router'
import { applyTheme } from './theme'
import { initTelegram } from './telegram'

applyTheme()
initTelegram()

createApp(App).use(createPinia()).use(router).mount('#app')
