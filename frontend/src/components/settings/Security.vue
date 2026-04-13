<template>
    <div>
        <div v-if="settingsStore.settingsLoaded" class="my-4">
            <template v-if="!settingsStore.settings.disableAuth">
                <p>
                    {{ $t("Current User") }}: <strong>{{ auth.username }}</strong>
                    <button class="btn btn-danger ml-4 mr-2 mb-2" @click="auth.logout">{{ $t("Logout") }}</button>
                </p>

                <h5 class="my-4 settings-subheading">{{ $t("Change Password") }}</h5>
                <form class="mb-3" @submit.prevent="savePassword">
                    <div class="mb-3">
                        <label for="current-password" class="form-label">{{ $t("Current Password") }}</label>
                        <input id="current-password" v-model="password.currentPassword" type="password" class="form-control" autocomplete="current-password" required />
                    </div>
                    <div class="mb-3">
                        <label for="new-password" class="form-label">{{ $t("New Password") }}</label>
                        <input id="new-password" v-model="password.newPassword" type="password" class="form-control" autocomplete="off" required />
                    </div>
                    <div class="mb-3">
                        <label for="repeat-new-password" class="form-label">{{ $t("Repeat New Password") }}</label>
                        <input id="repeat-new-password" v-model="password.repeatNewPassword" type="password" class="form-control" :class="{ 'is-invalid': invalidPassword }" autocomplete="off" required />
                        <div v-if="invalidPassword" class="invalid-feedback">{{ $t("passwordNotMatchMsg") }}</div>
                    </div>
                    <div>
                        <button class="btn btn-primary" type="submit">{{ $t("Update Password") }}</button>
                    </div>
                </form>
            </template>

            <div class="my-4">
                <h5 class="my-4 settings-subheading">{{ $t("Advanced") }}</h5>
                <div class="mb-4">
                    <button v-if="settingsStore.settings.disableAuth" class="btn btn-outline-primary mr-2 mb-2" @click="enableAuth">{{ $t("Enable Auth") }}</button>
                    <button v-if="!settingsStore.settings.disableAuth" class="btn btn-primary mr-2 mb-2" @click="confirmDialog?.show()">{{ $t("Disable Auth") }}</button>
                </div>
            </div>
        </div>

        <Confirm ref="confirmDialog" btn-style="btn-danger" :yes-text="$t('I understand, please disable')" :no-text="$t('Leave')" @yes="disableAuth">
            <!-- eslint-disable-next-line vue/no-v-html -->
            <p v-html="$t('disableauth.message1')"></p>
            <!-- eslint-disable-next-line vue/no-v-html -->
            <p v-html="$t('disableauth.message2')"></p>
            <p>{{ $t("Please use this option carefully!") }}</p>
            <div class="mb-3">
                <label for="current-password2" class="form-label">{{ $t("Current Password") }}</label>
                <input id="current-password2" v-model="password.currentPassword" type="password" class="form-control" required />
            </div>
        </Confirm>
    </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch } from "vue";
import Confirm from "../Confirm.vue";
import { useSettingsStore } from "../../stores/settings";
import { useAuthStore } from "../../stores/auth";
import { useSocketStore } from "../../stores/socket";
import { useToastHelper } from "../../composables/useToastHelper";

const settingsStore = useSettingsStore();
const auth = useAuthStore();
const socketStore = useSocketStore();
const toast = useToastHelper();

const confirmDialog = ref<InstanceType<typeof Confirm> | null>(null);
const invalidPassword = ref(false);

const password = reactive({
    currentPassword: "",
    newPassword: "",
    repeatNewPassword: "",
});

watch(() => password.repeatNewPassword, () => {
    invalidPassword.value = false;
});
watch(() => password.newPassword, () => {
    invalidPassword.value = false;
});

function savePassword() {
    if (password.newPassword !== password.repeatNewPassword) {
        invalidPassword.value = true;
    } else {
        socketStore.getSocket().emit("changePassword", password, (res: import("../../types").SocketResponse) => {
            toast.toastRes(res);
            if (res.ok) {
                password.currentPassword = "";
                password.newPassword = "";
                password.repeatNewPassword = "";
            }
        });
    }
}

function disableAuth() {
    settingsStore.settings.disableAuth = true;
    settingsStore.saveSettings(() => {
        password.currentPassword = "";
        auth.username = null;
        socketStore.socketIO.token = "autoLogin";
    }, password.currentPassword);
}

function enableAuth() {
    settingsStore.settings.disableAuth = false;
    settingsStore.saveSettings();
    auth.storage().removeItem("token");
    location.reload();
}
</script>
