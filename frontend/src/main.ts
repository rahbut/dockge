// Dayjs init inside this, so it has to be the first import
import "./utils";

import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import { router } from "./router";
import { i18n } from "./i18n";
import { useSocketStore } from "./stores/socket";
import { useThemeStore } from "./stores/theme";
import { useLangStore } from "./stores/lang";
import { useLayoutStore } from "./stores/layout";

// Dependencies
import Toast, { POSITION } from "vue-toastification";
import "@xterm/xterm/lib/xterm.js";

// CSS
import "@fontsource/jetbrains-mono";
import "vue-toastification/dist/index.css";
import "@xterm/xterm/css/xterm.css";
import "./styles/main.css";

// Set Title
document.title = document.title + " - " + location.host;

const pinia = createPinia();
const app = createApp(App);

app.use(pinia);
app.use(Toast, { position: POSITION.BOTTOM_RIGHT, showCloseButtonOnHover: true });
app.use(router);
app.use(i18n);

// Initialise stores that need startup side-effects
useThemeStore().init();
useLayoutStore().init();
useLangStore().init();
useSocketStore().initSocketIO();

app.mount("#app");
