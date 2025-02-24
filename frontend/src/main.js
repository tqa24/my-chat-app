import { createApp } from 'vue'
import App from './App.vue'
import router from './router';
import store from './store';
import { marked } from 'marked';
import DOMPurify from 'dompurify';

const app = createApp(App);
app.use(router);
app.use(store);

// Load user from localStorage if available
const savedUser = localStorage.getItem('user');
if (savedUser) {
    store.commit('setUser', JSON.parse(savedUser));
}

const savedUnreadCounts = localStorage.getItem('unreadCounts');
if (savedUnreadCounts) {
    store.state.unreadCounts = JSON.parse(savedUnreadCounts);
}

// --- Custom Directive (v-markdown) ---
app.directive('markdown', {
    // `mounted` is called when the bound element is inserted into the DOM
    mounted(el, binding) {
        // Use marked.parse() to render the Markdown
        const renderedHTML = marked.parse(binding.value || '', {
            gfm: true,          // Enable GitHub Flavored Markdown
            breaks: true,       // Use GFM line breaks (requires gfm: true)
            pedantic: false,    // Don't be strict about parsing
            sanitize: false,    //  Set to false, sanitize *after* parsing
            smartLists: true,   // Use smarter list behavior
            smartypants: true,  // Use "smart" typographic punctuation
        });
        // Sanitize the rendered HTML *before* inserting it
        el.innerHTML = DOMPurify.sanitize(renderedHTML);

    },
    // `updated` is called when the containing component has updated *and* its children updated.
    updated(el, binding) {
        if (binding.value !== binding.oldValue) {
            const renderedHTML = marked.parse(binding.value || '', {
                gfm: true,
                breaks: true,
                pedantic: false,
                sanitize: false, // Sanitize *after* parsing
                smartLists: true,
                smartypants: true,
            });
            el.innerHTML = DOMPurify.sanitize(renderedHTML);
        }
    },
});

app.mount('#app')