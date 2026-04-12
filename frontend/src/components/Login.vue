<template>
    <div class="flex items-center pt-10 pb-10">
        <div class="w-full max-w-[330px] p-4 mx-auto text-center">
            <form @submit.prevent="submit">
                <h1 class="text-xl mb-3 font-normal" />

                <input v-if="!tokenRequired" v-model="username" type="text" class="form-control" :placeholder="$t('Username')" autocomplete="username" required>

                <input v-if="!tokenRequired" v-model="password" type="password" class="form-control mt-3" :placeholder="$t('Password')" autocomplete="current-password" required>

                <input v-if="tokenRequired" v-model="token" type="text" maxlength="6" class="form-control mt-3" :placeholder="$t('Token')" autocomplete="one-time-code" required>

                <div class="flex justify-center mt-3 mb-3">
                    <div class="form-check">
                        <input id="remember" v-model="$root.remember" type="checkbox" value="remember-me" class="form-check-input">
                        <label class="form-check-label" for="remember">
                            {{ $t("Remember me") }}
                        </label>
                    </div>
                </div>

                <button class="w-full btn btn-primary" type="submit" :disabled="processing">
                    {{ $t("Login") }}
                </button>

                <div v-if="res && !res.ok" class="alert alert-danger mt-3" role="alert">
                    {{ $t(res.msg) }}
                </div>
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
            token: "",
            res: null,
            tokenRequired: false,
        };
    },

    mounted() {
        document.title += " - Login";
    },

    unmounted() {
        document.title = document.title.replace(" - Login", "");
    },

    methods: {
        submit() {
            this.processing = true;

            this.$root.login(this.username, this.password, this.token, (res) => {
                this.processing = false;

                if (res.tokenRequired) {
                    this.tokenRequired = true;
                } else {
                    this.res = res;
                }
            });
        },
    },
};
</script>
