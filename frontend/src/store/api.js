import axios from 'axios';
import router from '../router/index.js';
import store from './index.js';

const api = axios.create({
    baseURL: '/api',
});

// Add a request interceptor
api.interceptors.request.use(
    config => {
        const token = localStorage.getItem('token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    error => {
        return Promise.reject(error);
    }
);

api.interceptors.response.use(
    response => response,
    error => {
        if (error.response && error.response.status === 401) {
            // Token expired or invalid
            store.dispatch('logout');
            router.push('/login');
        }
        return Promise.reject(error);
    }
);

export default api;
