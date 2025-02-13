import { createStore } from 'vuex';
import axios from 'axios';
export default createStore({
    state: {
        user: null, // Store user information
        messages: [], // Store chat messages
        usersOnline: [], //Online user
        typingUsers: [], // Type user
        ws: null, // Add WebSocket
        selectedGroup: null,
        unreadCounts: {}, // Store unread counts for users and groups
    },
    mutations: {
        setUser(state, user) {
            state.user = user;
            if(user){
                localStorage.setItem('user', JSON.stringify(user)); // Persist user
            } else {
                localStorage.removeItem('user')
            }
        },
        setMessages(state, messages) {
            state.messages = messages;
        },
        addMessage(state, message) {
            state.messages.unshift(message); // Add to the beginning for newest-first
        },
        setUsersOnline(state, users) {
            state.usersOnline = users;
        },
        addTypingUser(state, username) {
            if (!state.typingUsers.includes(username)) {
                state.typingUsers.push(username);
            }
        },
        removeTypingUser(state, username) {
            state.typingUsers = state.typingUsers.filter(u => u !== username);
        },
        clearMessages(state) {
            state.messages = [];
        },
        setWs(state, wsInstance) {
            state.ws = wsInstance;
        },
        setSelectedGroup(state, group) {
            state.selectedGroup = group;
        },
        // --- New Mutations for Unread Counts ---
        setUnreadCount(state, { id, count }) {
            state.unreadCounts = { ...state.unreadCounts, [id]: count };
            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        incrementUnreadCount(state, id) {
            const currentCount = state.unreadCounts[id] || 0;
            state.unreadCounts = { ...state.unreadCounts, [id]: currentCount + 1 };
            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        clearUnreadCount(state, id) {
            state.unreadCounts = { ...state.unreadCounts, [id]: 0 };
            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
    },
    actions: {
        login({ commit }, user) {
            commit('setUser', user);
            // Set the Authorization header after successful login
            const token = localStorage.getItem('token');
            if (token) {
                axios.defaults.headers.common['Authorization'] = `Bearer ${token}`;
            }
        },
        logout({ commit }) {
            // Clear the Authorization header on logout
            delete axios.defaults.headers.common['Authorization'];
            commit('setUser', null);
            localStorage.removeItem('token'); // Remove the token on logout
            //Close websocket
            if (this.state.ws) {
                this.state.ws.close();
            }
        },
        setMessages({ commit }, messages) {
            commit('setMessages', messages);
        },
        addMessage({commit}, message){
            commit('addMessage', message)
        },
        setUsersOnline({ commit }, users) {
            commit('setUsersOnline', users);
        },
        addTypingUser({ commit }, username) {
            commit('addTypingUser', username);
        },
        removeTypingUser({ commit }, username) {
            commit('removeTypingUser', username);
        },
        clearMessages({commit}){
            commit('clearMessages')
        },
        setWs({ commit }, wsInstance) {
            commit('setWs', wsInstance);
        },
        setSelectedGroup({ commit }, group) {
            commit('setSelectedGroup', group);
        },
        // --- New Actions for Unread Counts ---
        incrementUnreadCount({ commit }, id) {
            commit('incrementUnreadCount', id);
        },
        markAsRead({ commit }, id) {
            commit('clearUnreadCount', id);
        },
    },
    getters: {
        currentUser: state => state.user,
        allMessages: state => state.messages,
        getUsersOnline: state => state.usersOnline,
        typingUsers: state => state.typingUsers,
        getUnreadCount: state => id => {
            return state.unreadCounts[id] || 0; // Return count or 0 if not found
        },
    },
});