import { i18n } from "../i18n";

// vue-i18n's Composition API types don't expose availableLocales / messages
// directly on the global instance — a cast is required here as a workaround.
type I18nGlobalCompat = {
    availableLocales: string[];
    messages: { value: Record<string, { languageName?: string }> };
};

/**
 * Returns the list of available locale codes and a helper to get the
 * human-readable language name for each.
 *
 * Centralises the vue-i18n Composition API workaround that is otherwise
 * copied across Setup.vue, General.vue, and Appearance.vue.
 */
export function useLocales() {
    const global = i18n.global as unknown as I18nGlobalCompat;

    const availableLocales: string[] = global.availableLocales ?? [];

    function localeLabel(lang: string): string {
        return global.messages?.value?.[lang]?.languageName ?? lang;
    }

    return { availableLocales, localeLabel };
}
