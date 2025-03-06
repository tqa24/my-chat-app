import { createStore } from 'vuex';
import api from './api';

export default createStore({
    state: {
        user: null,
        token:null,
        messages: [],
        usersOnline: [],
        typingUsers: [],
        ws: null,  // WebSocket instance
        selectedGroup: null,
        unreadCounts: {},
        replyingTo: null,
        groupMembers: [], // Store group members
    },
    mutations: {
        setUser(state, user) {
            state.user = user;
            if(user){
                localStorage.setItem('user', JSON.stringify(user));
            } else {
                localStorage.removeItem('user')
            }
        },
        setToken(state, token) {
            state.token = token;
            if(token){
                localStorage.setItem('token', token);
                // Set token in axios default headers
                api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
            } else {
                localStorage.removeItem('token');
                delete api.defaults.headers.common['Authorization'];
            }
        },
        setMessages(state, messages) {
            state.messages = messages;
        },
        addMessage(state, message) {
            if (message.reply_to_message_id) {
                const originalMessageIndex = state.messages.findIndex(m => m.id === message.reply_to_message_id);
                if (originalMessageIndex !== -1) {
                    message.reply_to_message = {
                        id: message.reply_to_message_id,
                        content: state.messages[originalMessageIndex].content,
                        sender_id: state.messages[originalMessageIndex].sender_id
                    };
                }
            }
            state.messages.push(message);
        },
        addMessages(state, newMessages) {
            const uniqueMessages = newMessages.filter(
                newMsg => !state.messages.some(
                    existingMsg => existingMsg.id === newMsg.id
                )
            );
            state.messages = [...uniqueMessages, ...state.messages];
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
        setUnreadCount(state, { id, count }) {
            state.unreadCounts = { ...state.unreadCounts, [id]: count };
            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        incrementUnreadCount(state, id) {
            const stringId = id?.toString();
            if (!stringId) {
                console.warn('Attempted to increment unread count with invalid ID:', id);
                return;
            }
            state.unreadCounts = {
                ...state.unreadCounts,
                [stringId]: (state.unreadCounts[stringId] || 0) + 1
            };
            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        clearUnreadCount(state, id) {
            const stringId = id?.toString();
            if (!stringId) {
                console.warn('Attempted to clear unread count with invalid ID:', id);
                return;
            }
            state.unreadCounts = {
                ...state.unreadCounts,
                [stringId]: 0
            };
            localStorage.setItem('unreadCounts', JSON.stringify(state.unreadCounts));
        },
        initializeUnreadCounts(state) {
            const saved = localStorage.getItem('unreadCounts');
            if (saved) {
                state.unreadCounts = JSON.parse(saved);
            }
        },
        setReplyingTo(state, message) {
            state.replyingTo = message;
        },
        toggleReaction(state, { messageId, reaction, add }) {
            const messageIndex = state.messages.findIndex(m => m.id === messageId);
            if (messageIndex === -1) return;
            const newMessage = { ...state.messages[messageIndex] };
            const newReactions = newMessage.reactions ? { ...newMessage.reactions } : {};

            if (add) {
                if (!newReactions[reaction]) {
                    newReactions[reaction] = [];
                }
                if (!newReactions[reaction].includes(state.user.id)) {
                    newReactions[reaction].push(state.user.id);
                }
            } else {
                if (newReactions[reaction]) {
                    newReactions[reaction] = newReactions[reaction].filter(id => id !== state.user.id);
                    if (newReactions[reaction].length === 0) {
                        delete newReactions[reaction];
                    }
                }
            }

            newMessage.reactions = newReactions;
            state.messages.splice(messageIndex, 1, newMessage);
        },
        updateMessageStatus(state, { messageId, status }) {
            const messageIndex = state.messages.findIndex(m => m.id === messageId);
            if (messageIndex !== -1) {
                state.messages[messageIndex].status = status;
            }
        },
        updateReaction(state, { messageId, userId, emoji, type }) {
            const messageIndex = state.messages.findIndex(m => m.id === messageId);
            if (messageIndex === -1) return;

            const message = { ...state.messages[messageIndex] };
            if (!message.reactions) {
                message.reactions = {};
            }

            if (type === "reaction_added") {
                if (!message.reactions[emoji]) {
                    message.reactions[emoji] = [];
                }
                if (!message.reactions[emoji].includes(userId)) {
                    message.reactions[emoji].push(userId);
                }
            } else if (type === "reaction_removed") {
                if (message.reactions[emoji]) {
                    const index = message.reactions[emoji].indexOf(userId);
                    if (index > -1) {
                        message.reactions[emoji].splice(index, 1);
                        if (message.reactions[emoji].length === 0) {
                            delete message.reactions[emoji];
                        }
                    }
                }
            }

            state.messages.splice(messageIndex, 1, message);
        },
        setGroupMembers(state, members) {
            state.groupMembers = members;
        },
        clearGroupMembers(state) {
            state.groupMembers = [];
        },
    },
    actions: {
        login({ commit }, { user, token }) {
            commit('setUser', user);
            commit('setToken', token);
            api.defaults.headers.common['Authorization'] = `Bearer ${token}`;
        },
        logout({ commit, state }) { // Add state to access ws
            delete api.defaults.headers.common['Authorization'];
            commit('setUser', null);
            commit('setToken', null);
            localStorage.removeItem('token');

            // Close the WebSocket connection on logout
            if (state.ws) {
                state.ws.close();
                commit('setWs', null); // Set ws to null in the store
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
        incrementUnreadCount({ commit }, id) {
            commit('incrementUnreadCount', id);
        },
        markAsRead({ commit }, id) {
            commit('clearUnreadCount', id);
        },
        initializeUnreadCounts({ commit }) {
            commit('initializeUnreadCounts');
        },
        async fetchGroupMessages({dispatch, state}){
            if(state.selectedGroup && state.selectedGroup.id){
                try {
                    const response = await api.get(
                        `/groups/${state.selectedGroup.id}/messages?page=1&pageSize=20`
                    );
                    dispatch('setMessages', response.data.messages.reverse());

                } catch (error) {
                    console.error("Failed to fetch group messages:", error);
                }
            }
        },
        async fetchMessages({dispatch, state}){
            if(state.selectedUser){
                try {
                    const response = await api.get(
                        `/messages?user1=${state.user?.id}&user2=${state.selectedUser?.id}&page=1&pageSize=20`
                    );
                    dispatch('setMessages', response.data.messages.reverse());
                } catch (error) {
                    console.error("Failed to fetch messages:", error);
                }
            }
        },
        toggleReaction({ commit }, payload) {
            commit('toggleReaction', payload);
        },
        updateMessageStatus({ commit }, { messageId, status }) {
            commit('updateMessageStatus', { messageId, status });
        },
        updateReaction({ commit }, payload) {
            commit('updateReaction', payload);
        },
        async fetchGroupMembers({ commit }, groupID) {
            try {
                // Use instance
                const response = await api.get(`/groups/${groupID}/members`);
                commit('setGroupMembers', response.data);
            } catch (error) {
                console.error("Failed to fetch group members:", error);
                // Optionally, commit an error to the store
            }
        },
        clearGroupMembers({ commit }) {
            commit('clearGroupMembers');
        },
    },
    getters: {
        currentUser: state => state.user,
        allMessages: state => {
            // Make sure to include messages where AI is either sender or receiver
            return state.messages.filter(message => {
                const AIUserID = "00000000-0000-0000-0000-000000000000";

                if (state.selectedUser && state.selectedUser.id === AIUserID) {
                    // When chatting with AI, show messages between current user and AI
                    return (message.sender_id === state.user?.id && message.receiver_id === AIUserID) ||
                        (message.sender_id === AIUserID && message.receiver_id === state.user?.id);
                }

                // Handle other message filtering...
                return true;
            });
        },
        getUsersOnline: state => state.usersOnline,
        typingUsers: state => state.typingUsers,
        getUnreadCount: (state) => (id) => {
            const stringId = id?.toString();
            const count = stringId ? (state.unreadCounts[stringId] || 0) : 0;
            return count;
        },
        getUserById: (state) => (userId) => {
            return state.usersOnline.find(user => user.id === userId);
        },
        groupMembers: state => state.groupMembers,
    },
});