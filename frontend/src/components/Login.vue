<template>
    <div class="flex items-center pt-10 pb-10">
        <div class="w-full max-w-[330px] p-4 mx-auto text-center">
            <form @submit.prevent="submit">
                <h1 class="text-xl mb-3 font-normal" />

                <input v-if="!tokenRequired" v-model="username" type="text" class="form-control" :placeholder="$t('Username')" autocomplete="username" required />
                <input v-if="!tokenRequired" v-model="password" type="password" class="form-control mt-3" :placeholder="$t('Password')" autocomplete="current-password" required />
                <input v-if="tokenRequired" v-model="token" type="text" maxlength="6" class="form-control mt-3" :placeholder="$t('Token')" autocomplete="one-time-code" required />

                <div class="flex justify-center mt-3 mb-3">
                    <div class="form-check">
                        <input id="remember" v-model="auth.remember" type="checkbox" value="remember-me" class="form-check-input" />
                        <label class="form-check-label" for="remember">{{ $t("Remember me") }}</label>
                    </div>
                </div>

                <button class="w-full btn btn-primary" type="submit" :disabled="processing">{{ $t("Login") }}</button>

                <div v-if="loginRes && !loginRes.ok" class="alert alert-danger mt-3" role="alert">
                    {{ loginRes.msg ? $t(loginRes.msg) : "" }}
                </div>
            </form>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from "vue";
import { useAuthStore } from "../stores/auth";
import type { SocketResponse } from "../types";

const auth = useAuthStore();

const processing = ref(false);
const username = ref("");
const password = ref("");
const token = ref("");
const loginRes = ref<(SocketResponse & { tokenRequired?: boolean }) | null>(null);
const tokenRequired = ref(false);

onMounted(() => {
    document.title += " - Login";
});

onUnmounted(() => {
    document.title = document.title.replace(" - Login", "");
});

function submit() {
    processing.value = true;
    auth.login(username.value, password.value, token.value, (res) => {
        processing.value = false;
        if (res.tokenRequired) {
            tokenRequired.value = true;
        } else {
            loginRes.value = res;
        }
    });
}
</script>
