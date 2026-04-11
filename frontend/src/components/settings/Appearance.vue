<template>
    <div>
        <div class="my-4">
            <label for="language" class="form-label">{{ $t("Language") }}</label>
            <select id="language" v-model="$root.language" class="form-select">
                <option
                    v-for="(lang, i) in $i18n.availableLocales"
                    :key="`Lang${i}`"
                    :value="lang"
                >
                    {{ $i18n.messages[lang].languageName }}
                </option>
            </select>
        </div>

        <div class="my-4">
            <label class="form-label">{{ $t("Theme") }}</label>
            <div class="flex gap-2">
                <button
                    v-for="opt in themeOptions"
                    :key="opt.value"
                    class="btn btn-outline-primary"
                    :class="{ 'active': $root.userTheme === opt.value }"
                    @click="$root.userTheme = opt.value"
                >
                    {{ $t(opt.label) }}
                </button>
            </div>
        </div>
    </div>
</template>

<script>
export default {
    data() {
        return {
            themeOptions: [
                { value: "light", label: "Light" },
                { value: "dark", label: "Dark" },
                { value: "auto", label: "Auto" },
            ],
        };
    },
};
</script>
