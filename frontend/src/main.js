import { createApp } from 'vue'
import App from './App.vue'
import router from './router';
import store from './store';

const app = createApp(App);
app.use(router);
app.use(store);
// Load user from localStorage if available
const savedUser = localStorage.getItem('user');
if (savedUser) {
    store.commit('setUser', JSON.parse(savedUser));
}
app.mount('#app')