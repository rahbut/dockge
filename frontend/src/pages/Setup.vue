<template>
    <div class="flex items-center pt-10 pb-10" data-cy="setup-form">
        <div class="w-full max-w-[330px] p-4 mx-auto text-center">
            <form @submit.prevent="submit">
                <div>
                    <object width="64" height="64" data="/icon.svg" />
                    <div style="font-size: 28px; font-weight: bold; margin-top: 5px;">
                        Dockge
                    </div>
                </div>

                <p class="mt-3">{{ $t("Create your admin account") }}</p>

                <div class="form-floating mt-3">
                    <select id="language" v-model="$root.language" class="form-select">
                        <option v-for="(lang, i) in $i18n.availableLocales" :key="`Lang${i}`" :value="lang">
                            {{ $i18n.messages[lang].languageName }}
                        </option>
                    </select>
                    <label for="language" class="form-label">{{ $t("Language") }}</label>
                </div>

                <div class="form-floating mt-3">
                    <input id="floatingInput" v-model="username" type="text" class="form-control" :placeholder="$t('Username')" required data-cy="username-input">
                    <label for="floatingInput">{{ $t("Username") }}</label>
                </div>

                <div class="form-floating mt-3">
                    <input id="floatingPassword" v-model="password" type="password" class="form-control" :placeholder="$t('Password')" required data-cy="password-input">
                    <label for="floatingPassword">{{ $t("Password") }}</label>
                </div>

                <div class="form-floating mt-3">
                    <input id="repeat" v-model="repeatPassword" type="password" class="form-control" :placeholder="$t('Repeat Password')" required data-cy="password-repeat-input">
                    <label for="repeat">{{ $t("Repeat Password") }}</label>
                </div>

                <button class="w-full btn btn-primary mt-3" type="submit" :disabled="processing" data-cy="submit-setup-form">
                    {{ $t("Create") }}
                </button>
            </form>
        </div>
    </div>
</template>

<script>
export default {
    data() {
        return {
            processing: false,
            username: "",
            password: "",
            repeatPassword: "",
        };
    },
    watch: {},
    mounted() {
        this.$root.getSocket().emit("needSetup", (needSetup) => {
            if (!needSetup) {
                this.$router.push("/");
            }
        });
    },
    methods: {
        submit() {
            this.processing = true;

            if (this.password !== this.repeatPassword) {
                this.$root.toastError("PasswordsDoNotMatch");
                this.processing = false;
                return;
            }

            this.$root.getSocket().emit("setup", this.username, this.password, (res) => {
                this.processing = false;
                this.$root.toastRes(res);

                if (res.ok) {
                    this.processing = true;
                    this.$root.login(this.username, this.password, "", () => {
                        this.processing = false;
                        this.$router.push("/");
                    });
                }
            });
        },
    },
};
</script>
