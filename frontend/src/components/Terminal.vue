<template>
    <div class="terminal-wrapper" :class="{ 'fit-height': fitHeight }" @mouseenter="hovered = true" @mouseleave="hovered = false">
        <div class="shadow-box" :class="{ 'fit-height': fitHeight }">
            <div v-pre ref="terminalEl" class="main-terminal"></div>
        </div>
        <button
            v-if="mode === 'displayOnly'"
            class="copy-btn"
            :class="{ copied: copyState === 'copied', visible: hovered || copyState === 'copied' }"
            :title="copyState === 'copied' ? 'Copied!' : 'Copy log to clipboard'"
            @click="copyBufferToClipboard"
        >
            <svg v-if="copyState !== 'copied'" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect>
                <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path>
            </svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="20 6 9 17 4 12"></polyline>
            </svg>
        </button>
    </div>
</template>

<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted } from "vue";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { TERMINAL_COLS, TERMINAL_ROWS } from "../utils";
import { useSocketStore } from "../stores/socket";
import { useThemeStore } from "../stores/theme";
import { useToastHelper } from "../composables/useToastHelper";

const props = withDefaults(defineProps<{
    name: string;
    endpoint: string;
    stackName?: string;
    serviceName?: string;
    shell?: string;
    rows?: number;
    cols?: number;
    fitHeight?: boolean;
    mode?: "displayOnly" | "mainTerminal" | "interactive";
}>(), {
    stackName: "",
    serviceName: "",
    shell: "bash",
    rows: TERMINAL_ROWS,
    cols: TERMINAL_COLS,
    fitHeight: false,
    mode: "displayOnly",
});

const emit = defineEmits<{ (e: "has-data"): void }>();

const socketStore = useSocketStore();
const themeStore = useThemeStore();
const toast = useToastHelper();

const terminalEl = ref<HTMLElement | null>(null);
const copyState = ref<"idle" | "copied">("idle");
const hovered = ref(false);
let terminal: Terminal;
let terminalFitAddOn: FitAddon | null = null;
let visibilityObserver: IntersectionObserver | null = null;
let first = true;
let terminalInputBuffer = "";
let cursorPosition = 0;

const darkTermTheme = {
    background: "#000000",
    foreground: "#cccccc",
    cursor: "#cccccc",
    cursorAccent: "#000000",
    selectionBackground: "#264f78",
};
const lightTermTheme = {
    background: "#f6f8fa",
    foreground: "#24292e",
    cursor: "#24292e",
    cursorAccent: "#f6f8fa",
    selectionBackground: "#add6ff80",
    black: "#24292e",
    red: "#d73a49",
    green: "#22863a",
    yellow: "#b08800",
    blue: "#0366d6",
    magenta: "#6f42c1",
    cyan: "#1b7c83",
    white: "#6a737d",
    brightBlack: "#959da5",
    brightRed: "#cb2431",
    brightGreen: "#28a745",
    brightYellow: "#dbab09",
    brightBlue: "#2188ff",
    brightMagenta: "#8a63d2",
    brightCyan: "#3192aa",
    brightWhite: "#d1d5da",
};

watch(() => themeStore.isDark, (isDark) => {
    if (terminal) {
        terminal.options.theme = isDark ? darkTermTheme : lightTermTheme;
    }
});

onMounted(() => {
    const currentTheme = themeStore.isDark ? darkTermTheme : lightTermTheme;
    terminal = new Terminal({
        fontSize: 14,
        fontFamily: "'JetBrains Mono', monospace",
        cursorBlink: props.mode !== "displayOnly",
        cols: props.cols,
        rows: props.rows,
        theme: currentTheme,
    });

    if (props.mode === "mainTerminal") {
        mainTerminalConfig();
    } else if (props.mode === "interactive") {
        interactiveTerminalConfig();
    }

    terminal.open(terminalEl.value!);
    terminal.options.theme = currentTheme;
    terminal.focus();

    terminalEl.value!.addEventListener("contextmenu", handleContextMenu);

    terminal.onSelectionChange(() => {
        const sel = terminal.getSelection();
        if (sel) {
            copyToClipboard(sel);
        }
    });

    terminal.onCursorMove(() => {
        if (first) {
            emit("has-data");
            first = false;
        }
    });

    bind();

    if (props.mode === "mainTerminal") {
        socketStore.emitAgent(props.endpoint, "mainTerminal", props.name, (res: import("../types").SocketResponse) => {
            if (!res.ok) {
                toast.toastRes(res);
            }
        });
    } else if (props.mode === "interactive") {
        socketStore.emitAgent(props.endpoint, "interactiveTerminal", props.stackName, props.serviceName, props.shell, (res: import("../types").SocketResponse) => {
            if (!res.ok) {
                toast.toastRes(res);
            }
        });
    }

    updateTerminalSize();

    visibilityObserver = new IntersectionObserver((entries) => {
        if (entries[0].isIntersecting && terminal) {
            terminal.options.theme = themeStore.isDark ? darkTermTheme : lightTermTheme;
            if (terminalFitAddOn) {
                fitTerminal();
            }
        }
    });
    visibilityObserver.observe(terminalEl.value!);
});

onUnmounted(() => {
    visibilityObserver?.disconnect();
    window.removeEventListener("resize", onResizeEvent);
    socketStore.unbindTerminal(props.name);
    terminal?.dispose();
    terminalEl.value?.removeEventListener("contextmenu", handleContextMenu);
});

function bind(endpointOverride?: string, nameOverride?: string) {
    const ep = endpointOverride ?? props.endpoint;
    const n = nameOverride ?? props.name;
    if (n) {
        socketStore.unbindTerminal(n);
        socketStore.bindTerminal(ep, n, terminal);
    }
}

function updateTerminalSize() {
    if (!terminalFitAddOn) {
        terminalFitAddOn = new FitAddon();
        terminal.loadAddon(terminalFitAddOn);
        window.addEventListener("resize", onResizeEvent);
    }
    fitTerminal();
}

function fitTerminal() {
    if (!terminalFitAddOn) {
        return;
    }
    if (props.fitHeight) {
        terminalFitAddOn.fit();
    } else {
        const dims = terminalFitAddOn.proposeDimensions();
        if (dims) {
            const cols = Math.floor(dims.cols);
            const rows = Math.floor(props.rows);
            if (cols > 0 && rows > 0 && isFinite(cols) && isFinite(rows)) {
                terminal.resize(cols, rows);
            }
        }
    }
}

function onResizeEvent() {
    fitTerminal();
    socketStore.emitAgent(props.endpoint, "terminalResize", props.name, terminal.rows, terminal.cols);
}

function removeInput() {
    const afterLen = terminalInputBuffer.length - cursorPosition;
    terminal.write(" ".repeat(afterLen) + "\b \b".repeat(terminalInputBuffer.length));
    cursorPosition = 0;
    terminalInputBuffer = "";
}

function clearCurrentLine() {
    terminal.write("\b".repeat(cursorPosition) + " ".repeat(terminalInputBuffer.length) + "\b".repeat(terminalInputBuffer.length));
}

function mainTerminalConfig() {
    terminal.onKey(e => {
        if (e.key === "\r") {
            if (!terminalInputBuffer.length) {
                return;
            }
            const buf = terminalInputBuffer;
            removeInput();
            socketStore.emitAgent(props.endpoint, "terminalInput", props.name, buf + e.key, (err: { msg?: string }) => {
                toast.toastError(err.msg ?? "");
            });
        } else if (e.key === "\u007F") { // Backspace
            if (cursorPosition > 0) {
                const before = terminalInputBuffer.slice(0, cursorPosition - 1);
                const after = terminalInputBuffer.slice(cursorPosition);
                terminalInputBuffer = before + after;
                cursorPosition--;
                terminal.write("\b" + after + " \b".repeat(after.length + 1));
            }
        } else if (e.key === "\u001B\u005B\u0033\u007E") { // Delete
            if (cursorPosition < terminalInputBuffer.length) {
                const before = terminalInputBuffer.slice(0, cursorPosition);
                const after = terminalInputBuffer.slice(cursorPosition + 1);
                terminalInputBuffer = before + after;
                terminal.write(after + " \b".repeat(after.length + 1));
            }
        } else if (e.key === "\u001B\u005B\u0041" || e.key === "\u001B\u005B\u0042") { // UP/DOWN
            // ignore
        } else if (e.key === "\u001B\u005B\u0043") { // RIGHT
            if (cursorPosition < terminalInputBuffer.length) {
                terminal.write(terminalInputBuffer[cursorPosition]);
                cursorPosition++;
            }
        } else if (e.key === "\u001B\u005B\u0044") { // LEFT
            if (cursorPosition > 0) {
                terminal.write("\b");
                cursorPosition--;
            }
        } else if (e.key === "\u0003") { // Ctrl+C
            socketStore.emitAgent(props.endpoint, "terminalInput", props.name, e.key);
            removeInput();
        } else if (e.key === "\u0016" || (e.domEvent?.ctrlKey && e.key.toLowerCase() === "v")) {
            handlePaste();
        } else if (e.key === "\u0009" || e.key.startsWith("\u001B")) {
            // ignore tab / other special
        } else {
            const before = terminalInputBuffer.slice(0, cursorPosition);
            const after = terminalInputBuffer.slice(cursorPosition);
            terminalInputBuffer = before + e.key + after;
            terminal.write(e.key + after + "\b".repeat(after.length));
            cursorPosition++;
        }
    });
}

function interactiveTerminalConfig() {
    terminal.onKey(e => {
        if (e.key === "\u0016" || (e.domEvent?.ctrlKey && e.key.toLowerCase() === "v")) {
            handlePaste();
            return;
        }
        socketStore.emitAgent(props.endpoint, "terminalInput", props.name, e.key, (res: import("../types").SocketResponse) => {
            if (!res.ok) {
                toast.toastRes(res);
            }
        });
    });
}

async function handlePaste() {
    try {
        const text = await navigator.clipboard.readText();
        if (text) {
            pasteText(text);
        }
    } catch (e) {
        console.error("Clipboard read failed", e);
    }
}

function pasteText(text: string) {
    if (props.mode === "mainTerminal") {
        const before = terminalInputBuffer.slice(0, cursorPosition);
        const after = terminalInputBuffer.slice(cursorPosition);
        terminalInputBuffer = before + text + after;
        clearCurrentLine();
        terminal.write(terminalInputBuffer);
        cursorPosition += text.length;
        terminal.write("\b".repeat(after.length));
    } else if (props.mode === "interactive") {
        socketStore.emitAgent(props.endpoint, "terminalInput", props.name, text, (res: import("../types").SocketResponse) => {
            if (!res.ok) {
                toast.toastRes(res);
            }
        });
    }
}

function handleContextMenu(event: Event) {
    event.preventDefault();
    if (props.mode === "mainTerminal" || props.mode === "interactive") {
        handlePaste();
    }
}

async function copyToClipboard(text: string) {
    try {
        await navigator.clipboard.writeText(text);
    } catch (e) {
        console.error(e);
    }
}

async function copyBufferToClipboard() {
    const buf = terminal.buffer.active;
    const lines: string[] = [];
    for (let i = 0; i < buf.length; i++) {
        lines.push(buf.getLine(i)?.translateToString(true) ?? "");
    }
    // Strip trailing blank lines
    while (lines.length && lines[lines.length - 1].trim() === "") {
        lines.pop();
    }
    await copyToClipboard(lines.join("\n"));
    copyState.value = "copied";
    setTimeout(() => {
        copyState.value = "idle";
    }, 2000);
}

defineExpose({ bind });
</script>

<style scoped>
.main-terminal { overflow: hidden; }
.fit-height .main-terminal { height: 100%; }

.terminal-wrapper { position: relative; }
.fit-height.terminal-wrapper { height: 100%; }

.copy-btn {
    position: absolute;
    top: 0.35rem;
    right: 0.4rem;
    z-index: 100;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.6rem;
    height: 1.6rem;
    padding: 0;
    border: 1px solid transparent;
    border-radius: 4px;
    background: transparent;
    color: #6a737d;
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, color 0.15s, background 0.15s;
}

.copy-btn.visible,
.copy-btn:focus-visible {
    opacity: 1;
}

.copy-btn:hover {
    background: rgba(0, 0, 0, 0.08);
    color: #24292e;
    border-color: rgba(0, 0, 0, 0.1);
}

.dark .copy-btn:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #cccccc;
    border-color: rgba(255, 255, 255, 0.15);
}

.copy-btn.copied {
    opacity: 1;
    color: #28a745;
}
</style>

<style>
.terminal { padding: 0; background-color: #f6f8fa; }
.dark .terminal { background-color: #000000 !important; }
.xterm .xterm-viewport, .xterm .xterm-screen { background-color: #f6f8fa !important; }
.dark .xterm .xterm-viewport, .dark .xterm .xterm-screen { background-color: #000000 !important; }
</style>
