import { createRouter, createWebHistory } from "vue-router";

// Eagerly load the shell components — they're always needed
import Layout from "./layouts/Layout.vue";
import Dashboard from "./pages/Dashboard.vue";
import DashboardHome from "./pages/DashboardHome.vue";

// Lazy-load everything else for better code splitting
const Setup = () => import("./pages/Setup.vue");
const Compose = () => import("./pages/Compose.vue");
const Console = () => import("./pages/Console.vue");
const ContainerTerminal = () => import("./pages/ContainerTerminal.vue");
const Settings = () => import("./pages/Settings.vue");

// Settings sub-pages — all lazy
const General = () => import("./components/settings/General.vue");
const Appearance = () => import("./components/settings/Appearance.vue");
const Security = () => import("./components/settings/Security.vue");
const GlobalEnv = () => import("./components/settings/GlobalEnv.vue");
const About = () => import("./components/settings/About.vue");

const routes = [
    {
        path: "/empty",
        component: Layout,
        children: [
            {
                path: "",
                component: Dashboard,
                meta: { requiresAuth: true },
                children: [
                    {
                        name: "DashboardHome",
                        path: "/",
                        component: DashboardHome,
                        children: [
                            { path: "/compose", component: Compose },
                            { path: "/compose/:stackName/:endpoint", component: Compose },
                            { path: "/compose/:stackName", component: Compose },
                            { path: "/terminal/:stackName/:serviceName/:type", component: ContainerTerminal, name: "containerTerminal" },
                            { path: "/terminal/:stackName/:serviceName/:type/:endpoint", component: ContainerTerminal, name: "containerTerminalEndpoint" },
                        ],
                    },
                    { path: "/console", component: Console },
                    { path: "/console/:endpoint", component: Console },
                    {
                        path: "/settings",
                        component: Settings,
                        children: [
                            { path: "general", component: General },
                            { path: "appearance", component: Appearance },
                            { path: "security", component: Security },
                            { path: "globalEnv", component: GlobalEnv },
                            { path: "about", component: About },
                        ],
                    },
                ],
            },
        ],
    },
    {
        path: "/setup",
        component: Setup,
    },
];

export const router = createRouter({
    linkActiveClass: "active",
    history: createWebHistory(),
    routes,
});

// Auth guard — enforces authentication on all routes marked requiresAuth.
// The socket connection is async, so we only block navigation when the socket
// is confirmed connected (allowLoginDialog = true) but the user is not logged in.
// During the initial socket handshake we allow navigation through so Layout.vue
// can render the connecting/login state correctly.
router.beforeEach((to) => {
    if (!to.meta.requiresAuth) {
        return true;
    }

    return import("./stores/auth").then(({ useAuthStore }) => {
        const auth = useAuthStore();
        // Not yet connected — let Layout handle the connecting state
        if (!auth.allowLoginDialog && !auth.loggedIn) {
            return true;
        }
        // Logged in — allow through
        if (auth.loggedIn) {
            return true;
        }
        // Connected + login dialog shown but not logged in — redirect to root
        // where Layout.vue will present the Login component
        return { path: "/" };
    });
});
