<template>
    <div class="flex items-center pt-10 pb-10" data-cy="setup-form">
        <div class="w-full max-w-[330px] p-4 mx-auto text-center">
            <form @submit.prevent="submit">
                <div>
                    <object width="64" height="64" data="/icon.svg" />
                    <div style="font-size:28px;font-weight:bold;margin-top:5px">Dockge</div>
                </div>
                <p class="mt-3">{{ $t("Create your admin account") }}</p>

                <select v-model="langStore.language" class="form-select mt-3">
                    <option v-for="(lang, i) in availableLocales" :key="`Lang${i}`" :value="lang">
                        {{ localeLabel(lang) }}
                    </option>
                </select>

                <input v-model="username" type="text" class="form-control mt-3" :placeholder="$t('Username')" required data-cy="username-input" />
                <input v-model="password" type="password" class="form-control mt-3" :placeholder="$t('Password')" required data-cy="password-input" />
                <input v-model="repeatPassword" type="password" class="form-control mt-3" :placeholder="$t('Repeat Password')" required data-cy="password-repeat-input" />

                <button class="w-full btn btn-primary mt-3" type="submit" :disabled="processing" data-cy="submit-setup-form">
                    {{ $t("Create") }}
                </button>
            </form>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { useRouter } from "vue-router";
import { useSocketStore } from "../stores/socket";
import { useAuthStore } from "../stores/auth";
import { useLangStore } from "../stores/lang";
import { useToastHelper } from "../composables/useToastHelper";
import { useLocales } from "../composables/useLocales";

const router = useRouter();
const socketStore = useSocketStore();
const auth = useAuthStore();
const langStore = useLangStore();
const toast = useToastHelper();
const { availableLocales, localeLabel } = useLocales();

const processing = ref(false);
const username = ref("");
const password = ref("");
const repeatPassword = ref("");

onMounted(() => {
    socketStore.getSocket().emit("needSetup", (needSetup: boolean) => {
        if (!needSetup) {
            router.push("/");
        }
    });
});

function submit() {
    processing.value = true;
    if (password.value !== repeatPassword.value) {
        toast.toastError("PasswordsDoNotMatch");
        processing.value = false;
        return;
    }
    socketStore.getSocket().emit("setup", username.value, password.value, (res: import("../types").SocketResponse) => {
        processing.value = false;
        toast.toastRes(res);
        if (res.ok) {
            processing.value = true;
            auth.login(username.value, password.value, "", () => {
                processing.value = false;
                router.push("/");
            });
        }
    });
}
</script>
