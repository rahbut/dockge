import { defineStore } from "pinia";
import { ref, computed, watch } from "vue";
import jwtDecode from "jwt-decode";
import { useSocketStore } from "./socket";
import type { SocketResponse } from "../types";

type JWTPayload = Record<string, unknown>;
type LoginCallback = (res: SocketResponse & { tokenRequired?: boolean }) => void;

export const useAuthStore = defineStore("auth", () => {
    const loggedIn = ref(false);
    const allowLoginDialog = ref(false);
    const username = ref<string | null>(null);
    const remember = ref(localStorage.getItem("remember") !== "0");

    const usernameFirstChar = computed(() => {
        if (typeof username.value === "string" && username.value.length >= 1) {
            return username.value.charAt(0).toUpperCase();
        }
        return "🐬";
    });

    watch(remember, (val) => {
        localStorage.setItem("remember", val ? "1" : "0");
    });

    function storage(): Storage {
        return remember.value ? localStorage : sessionStorage;
    }

    function getJWTPayload(): JWTPayload | undefined {
        const token = storage().getItem("token");
        if (token && token !== "autoLogin") {
            try {
                return jwtDecode(token) as JWTPayload;
            } catch {
                return undefined;
            }
        }
        return undefined;
    }

    function login(user: string, password: string, token: string, callback: LoginCallback) {
        const socketStore = useSocketStore();
        socketStore.getSocket().emit("login", { username: user, password, token }, (res: SocketResponse & { tokenRequired?: boolean; token?: string }) => {
            if (res.tokenRequired) {
                callback(res);
                return;
            }
            if (res.ok && res.token) {
                storage().setItem("token", res.token);
                socketStore.socketIO.token = res.token;
                loggedIn.value = true;
                username.value = (getJWTPayload()?.username as string) ?? null;
                socketStore.afterLogin();
                history.pushState({}, "");
            }
            callback(res);
        });
    }

    function loginByToken(token: string) {
        const socketStore = useSocketStore();
        socketStore.getSocket().emit("loginByToken", token, (res: SocketResponse) => {
            allowLoginDialog.value = true;
            if (!res.ok) {
                logout();
            } else {
                loggedIn.value = true;
                username.value = (getJWTPayload()?.username as string) ?? null;
                socketStore.afterLogin();
            }
        });
    }

    function logout() {
        const socketStore = useSocketStore();
        socketStore.getSocket().emit("logout", () => { });
        storage().removeItem("token");
        socketStore.socketIO.token = null;
        loggedIn.value = false;
        username.value = null;
    }

    return {
        loggedIn,
        allowLoginDialog,
        username,
        remember,
        usernameFirstChar,
        storage,
        getJWTPayload,
        login,
        loginByToken,
        logout,
    };
});
