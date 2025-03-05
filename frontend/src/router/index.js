import { createRouter, createWebHistory } from 'vue-router';
import HomeView from '../views/HomeView.vue';
import LoginView from '../views/LoginView.vue';
import RegisterView from '../views/RegisterView.vue';
import CreateGroup from '../components/CreateGroup.vue';
import JoinGroupByCode from '../components/JoinGroupByCode.vue';
import VerifyOTPForm from '../components/VerifyOTPForm.vue';

const routes = [
    {
        path: '/',
        name: 'home',
        component: HomeView,
        meta: { requiresAuth: true }, // Protect this route
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
    {
        path: '/join-group', // Add this route for joining by code
        name: 'join-group',
        component: JoinGroupByCode,
        meta: { requiresAuth: true },
    },
    {
        path: '/verify-otp', // Add the new route
        name: 'verify-otp',
        component: VerifyOTPForm,
    },
];

const router = createRouter({
    history: createWebHistory('/static/'), // Set base URL
    routes,
});

router.beforeEach((to, from, next) => {
    const isLoggedIn = localStorage.getItem('user');
    if (to.matched.some(record => record.meta.requiresAuth) && !isLoggedIn) {
        next('/login');
    } else {
        next();
    }
});

export default router;