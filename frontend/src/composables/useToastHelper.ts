import { useToast } from "vue-toastification";
import { useI18n } from "vue-i18n";
import type { SocketResponse } from "../types";

type MsgObject = { key: string; values?: Record<string, unknown> };
type ToastResponse = SocketResponse & { msg?: string | MsgObject };

/**
 * Composable wrapper around vue-toastification that mirrors the
 * toastRes / toastSuccess / toastError methods previously on $root.
 */
export function useToastHelper() {
    const toast = useToast();
    const { t } = useI18n();

    function toastRes(res: ToastResponse) {
        let msg: string | undefined = typeof res.msg === "string" ? res.msg : undefined;
        if (res.msgi18n) {
            const rawMsg = res.msg;
            if (rawMsg != null && typeof rawMsg === "object" && !Array.isArray(rawMsg)) {
                const msgObj = rawMsg as MsgObject;
                msg = t(msgObj.key, msgObj.values as Record<string, unknown>);
            } else if (typeof rawMsg === "string" && rawMsg) {
                msg = t(rawMsg);
            }
        }
        if (res.ok) {
            toast.success(msg);
        } else {
            toast.error(msg);
        }
    }

    function toastSuccess(msg: string) {
        toast.success(t(msg));
    }

    function toastError(msg: string) {
        toast.error(msg);
    }

    return { toastRes, toastSuccess, toastError };
}
