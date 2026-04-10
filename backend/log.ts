// Console colors
// https://stackoverflow.com/questions/9781218/how-to-change-node-jss-console-font-color
import { intHash, isDev } from "../common/util-common";
import dayjs from "dayjs";

export const consoleStyleReset = "\x1b[0m";
export const consoleStyleBright = "\x1b[1m";
export const consoleStyleDim = "\x1b[2m";
export const consoleStyleUnderscore = "\x1b[4m";
export const consoleStyleBlink = "\x1b[5m";
export const consoleStyleReverse = "\x1b[7m";
export const consoleStyleHidden = "\x1b[8m";

export const consoleStyleFgBlack = "\x1b[30m";
export const consoleStyleFgRed = "\x1b[31m";
export const consoleStyleFgGreen = "\x1b[32m";
export const consoleStyleFgYellow = "\x1b[33m";
export const consoleStyleFgBlue = "\x1b[34m";
export const consoleStyleFgMagenta = "\x1b[35m";
export const consoleStyleFgCyan = "\x1b[36m";
export const consoleStyleFgWhite = "\x1b[37m";
export const consoleStyleFgGray = "\x1b[90m";
export const consoleStyleFgOrange = "\x1b[38;5;208m";
export const consoleStyleFgLightGreen = "\x1b[38;5;119m";
export const consoleStyleFgLightBlue = "\x1b[38;5;117m";
export const consoleStyleFgViolet = "\x1b[38;5;141m";
export const consoleStyleFgBrown = "\x1b[38;5;130m";
export const consoleStyleFgPink = "\x1b[38;5;219m";

export const consoleStyleBgBlack = "\x1b[40m";
export const consoleStyleBgRed = "\x1b[41m";
export const consoleStyleBgGreen = "\x1b[42m";
export const consoleStyleBgYellow = "\x1b[43m";
export const consoleStyleBgBlue = "\x1b[44m";
export const consoleStyleBgMagenta = "\x1b[45m";
export const consoleStyleBgCyan = "\x1b[46m";
export const consoleStyleBgWhite = "\x1b[47m";
export const consoleStyleBgGray = "\x1b[100m";

const consoleModuleColors = [
    consoleStyleFgCyan,
    consoleStyleFgGreen,
    consoleStyleFgLightGreen,
    consoleStyleFgBlue,
    consoleStyleFgLightBlue,
    consoleStyleFgMagenta,
    consoleStyleFgOrange,
    consoleStyleFgViolet,
    consoleStyleFgBrown,
    consoleStyleFgPink,
];

const consoleLevelColors : Record<string, string> = {
    "INFO": consoleStyleFgCyan,
    "WARN": consoleStyleFgYellow,
    "ERROR": consoleStyleFgRed,
    "DEBUG": consoleStyleFgGray,
};

class Logger {

    /**
     * DOCKGE_HIDE_LOG=debug_monitor,info_monitor
     *
     * Example:
     *  [
     *     "debug_monitor",          // Hide all logs that level is debug and the module is monitor
     *     "info_monitor",
     *  ]
     */
    hideLog : Record<string, string[]> = {
        info: [],
        warn: [],
        error: [],
        debug: [],
    };

    /**
     *
     */
    constructor() {
        if (typeof process !== "undefined" && process.env.DOCKGE_HIDE_LOG) {
            const list = process.env.DOCKGE_HIDE_LOG.split(",").map(v => v.toLowerCase());

            for (const pair of list) {
                // split first "_" only
                const values = pair.split(/_(.*)/s);

                if (values.length >= 2) {
                    this.hideLog[values[0]].push(values[1]);
                }
            }

            this.debug("server", "DOCKGE_HIDE_LOG is set");
            this.debug("server", this.hideLog);
        }
    }

    /**
     * Write a message to the log
     * @param module The module the log comes from
     * @param msg Message to write
     * @param level Log level. One of INFO, WARN, ERROR, DEBUG or can be customized.
     */
    log(module: string, msg: unknown, level: string) {
        if (level === "DEBUG" && !isDev) {
            return;
        }

        if (this.hideLog[level] && this.hideLog[level].includes(module.toLowerCase())) {
            return;
        }

        module = module.toUpperCase();
        level = level.toUpperCase();

        let now;
        if (dayjs.tz) {
            now = dayjs.tz(new Date()).format();
        } else {
            now = dayjs().format();
        }

        const levelColor = consoleLevelColors[level];
        const moduleColor = consoleModuleColors[intHash(module, consoleModuleColors.length)];

        let timePart = consoleStyleFgCyan + now + consoleStyleReset;
        const modulePart = "[" + moduleColor + module + consoleStyleReset + "]";
        const levelPart = levelColor + `${level}:` + consoleStyleReset;

        if (level === "INFO") {
            console.info(timePart, modulePart, levelPart, msg);
        } else if (level === "WARN") {
            console.warn(timePart, modulePart, levelPart, msg);
        } else if (level === "ERROR") {
            let msgPart : unknown;
            if (typeof msg === "string") {
                msgPart = consoleStyleFgRed + msg + consoleStyleReset;
            } else {
                msgPart = msg;
            }
            console.error(timePart, modulePart, levelPart, msgPart);
        } else if (level === "DEBUG") {
            if (isDev) {
                timePart = consoleStyleFgGray + now + consoleStyleReset;
                let msgPart : unknown;
                if (typeof msg === "string") {
                    msgPart = consoleStyleFgGray + msg + consoleStyleReset;
                } else {
                    msgPart = msg;
                }
                console.debug(timePart, modulePart, levelPart, msgPart);
            }
        } else {
            console.log(timePart, modulePart, msg);
        }
    }

    /**
     * Log an INFO message
     * @param module Module log comes from
     * @param msg Message to write
     */
    info(module: string, msg: unknown) {
        this.log(module, msg, "info");
    }

    /**
     * Log a WARN message
     * @param module Module log comes from
     * @param msg Message to write
     */
    warn(module: string, msg: unknown) {
        this.log(module, msg, "warn");
    }

    /**
     * Log an ERROR message
     * @param module Module log comes from
     * @param msg Message to write
     */
    error(module: string, msg: unknown) {
        this.log(module, msg, "error");
    }

    /**
     * Log a DEBUG message
     * @param module Module log comes from
     * @param msg Message to write
     */
    debug(module: string, msg: unknown) {
        this.log(module, msg, "debug");
    }

    /**
     * Log an exception as an ERROR
     * @param module Module log comes from
     * @param exception The exception to include
     * @param msg The message to write
     */
    exception(module: string, exception: unknown, msg: unknown) {
        let finalMessage = exception;

        if (msg) {
            finalMessage = `${msg}: ${exception}`;
        }

        this.log(module, finalMessage, "error");
    }
}

export const log = new Logger();
