<template>
  <div id="app">
    <nav v-if="currentUser">
      <router-link to="/">Home</router-link> |
      <button @click="logout">Logout</button>
    </nav>
    <router-view/>
  </div>
</template>

<script>
import { computed } from 'vue';
import { useStore } from 'vuex';
import { useRouter } from 'vue-router';
export default {
  setup() {
    const store = useStore();
    const router = useRouter();
    const currentUser = computed(() => store.getters.currentUser);
    const logout = () => {
      store.dispatch('logout');
      router.push('/login')
    }

    return { currentUser, logout };
  }
}
</script>