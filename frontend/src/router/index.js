import { createRouter, createWebHistory } from 'vue-router';
import HomeView from '../views/HomeView.vue';
import LoginView from '../views/LoginView.vue';
import RegisterView from '../views/RegisterView.vue';
import CreateGroup from '../components/CreateGroup.vue'; // Import the component

const routes = [
    {
        path: '/',
        name: 'home',
        component: HomeView,
        meta: { requiresAuth: true },
    },
    {
        path: '/login',
        name: 'login',
        component: LoginView,
    },
    {
        path: '/register',
        name: 'register',
        component: RegisterView,
    },
    {
        path: '/create-group', // Add this route
        name: 'create-group',
        component: CreateGroup, // Use the component
        meta: { requiresAuth: true }, // Protect this route
    },
];

const router = createRouter({
    history: createWebHistory(),
    routes,
});

router.beforeEach((to, from, next) => {
    const token = localStorage.getItem('token');
    const isLoggedIn = !!token; // Convert token to boolean (true if exists, false if not)

    if (to.matched.some(record => record.meta.requiresAuth) && !isLoggedIn) {
        next('/login'); // Redirect to login if not authenticated
    } else {
        next();
    }
});
export default router;