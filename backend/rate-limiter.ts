// "limit" is bugged in Typescript, use "limiter-es6-compat" instead
// See https://github.com/jhurliman/node-rate-limiter/issues/80
import { RateLimiter, RateLimiterOpts } from "limiter-es6-compat";
import { log } from "./log";

export interface KumaRateLimiterOpts extends RateLimiterOpts {
    errorMessage : string;
}

export type KumaRateLimiterCallback = (err : object) => void;

interface PerIPEntry {
    limiter : RateLimiter;
    lastAccess : number;
}

const CLEANUP_INTERVAL_MS = 10 * 60 * 1000;
const STALE_THRESHOLD_MS = 10 * 60 * 1000;

class KumaRateLimiter {

    errorMessage : string;
    config : KumaRateLimiterOpts;
    limiters : Map<string, PerIPEntry> = new Map();

    /**
     * @param {object} config Rate limiter configuration object
     */
    constructor(config : KumaRateLimiterOpts) {
        this.errorMessage = config.errorMessage;
        this.config = config;

        setInterval(() => this.cleanup(), CLEANUP_INTERVAL_MS);
    }

    /**
     * Get or create a rate limiter for the given key
     * @param {string} key The key to look up (typically an IP address)
     * @returns {RateLimiter} The rate limiter for this key
     */
    getLimiter(key : string) : RateLimiter {
        let entry = this.limiters.get(key);
        if (!entry) {
            entry = {
                limiter: new RateLimiter(this.config),
                lastAccess: Date.now(),
            };
            this.limiters.set(key, entry);
        } else {
            entry.lastAccess = Date.now();
        }
        return entry.limiter;
    }

    /**
     * Should the request be passed through
     * @param callback Callback function to call with decision
     * @param {string} key The key to rate limit by (typically an IP address)
     * @param {number} num Number of tokens to remove
     * @returns {Promise<boolean>} Should the request be allowed?
     */
    async pass(callback : KumaRateLimiterCallback, key = "global", num = 1) {
        const limiter = this.getLimiter(key);
        const remainingRequests = await limiter.removeTokens(num);
        log.info("rate-limit", `remaining requests for ${key}: ${remainingRequests}`);
        if (remainingRequests < 0) {
            if (callback) {
                callback({
                    ok: false,
                    msg: this.errorMessage,
                });
            }
            return false;
        }
        return true;
    }

    /**
     * Remove stale rate limiter entries
     */
    cleanup() {
        const now = Date.now();
        for (const [ key, entry ] of this.limiters) {
            if (now - entry.lastAccess > STALE_THRESHOLD_MS) {
                this.limiters.delete(key);
            }
        }
    }
}

export const loginRateLimiter = new KumaRateLimiter({
    tokensPerInterval: 20,
    interval: "minute",
    fireImmediately: true,
    errorMessage: "Too frequently, try again later."
});

export const apiRateLimiter = new KumaRateLimiter({
    tokensPerInterval: 60,
    interval: "minute",
    fireImmediately: true,
    errorMessage: "Too frequently, try again later."
});

