<template>
    <transition name="slide-fade" appear>
        <div v-if="$route.name === 'DashboardHome'">
            <h1 class="mb-3">{{ $t("home") }}</h1>

            <div class="flex flex-wrap gap-4">
                <!-- Left -->
                <div class="w-full md:w-[calc(58%-0.5rem)]">
                    <!-- Stats -->
                    <div class="shadow-box big-padding text-center mb-4">
                        <div class="flex justify-around">
                            <div>
                                <h3>{{ $t("active") }}</h3>
                                <span class="num active">{{ activeNum }}</span>
                            </div>
                            <div>
                                <h3>{{ $t("exited") }}</h3>
                                <span class="num exited">{{ exitedNum }}</span>
                            </div>
                            <div>
                                <h3>{{ $t("inactive") }}</h3>
                                <span class="num inactive">{{ inactiveNum }}</span>
                            </div>
                        </div>
                    </div>

                    <!-- Maintenance actions -->
                    <div class="shadow-box big-padding mb-4">
                        <!-- Check for Updates -->
                        <div class="text-center mb-4 pb-4 border-b border-gray-200 dark:border-[#1d2634]">
                            <button class="btn btn-normal" :disabled="checkingUpdates" @click="checkAllUpdates">
                                <RefreshCwIcon :size="14" class="mr-1 inline" :class="{ 'animate-spin': checkingUpdates }" />
                                {{ checkingUpdates ? $t('checkingForUpdates') : $t('checkForUpdates') }}
                            </button>
                            <div v-if="updateCheckDone" class="mt-2">
                                <span v-if="updatesAvailableNum > 0" class="text-yellow-500 flex items-center justify-center gap-1">
                                    <ArrowUpCircleIcon :size="14" />
                                    {{ $t('stacksHaveUpdates', [updatesAvailableNum, totalCheckedNum]) }}
                                </span>
                                <span v-else class="text-green-500 flex items-center justify-center gap-1">
                                    <CheckCircleIcon :size="14" />
                                    {{ $t('noUpdatesFound') }}
                                </span>
                            </div>
                        </div>

                        <!-- Prune Images -->
                        <div class="text-center">
                            <div class="inline-flex btn-group">
                                <button class="btn btn-normal" :disabled="pruning" @click="pruneImages(false)">
                                    <Trash2Icon :size="14" class="mr-1 inline" :class="{ 'animate-spin': pruning }" />
                                    {{ pruning ? $t('pruningImages') : $t('pruneImages') }}
                                </button>
                                <HMenu as="div" class="relative">
                                    <HMenuButton class="btn btn-normal rounded-l-none px-3 self-stretch flex items-center" :disabled="pruning">
                                        <ChevronDownIcon :size="14" />
                                    </HMenuButton>
                                    <transition
                                        enter-active-class="transition duration-100 ease-out"
                                        enter-from-class="transform scale-95 opacity-0"
                                        enter-to-class="transform scale-100 opacity-1"
                                        leave-active-class="transition duration-75 ease-in"
                                        leave-from-class="transform scale-100 opacity-1"
                                        leave-to-class="transform scale-95 opacity-0"
                                    >
                                        <HMenuItems class="absolute right-0 mt-1 w-52 origin-top-right rounded-xl overflow-hidden shadow-xl bg-white dark:bg-[#0d1117] border border-gray-100 dark:border-[#1d2634] z-50 focus:outline-none">
                                            <HMenuItem v-slot="{ active: itemActive }">
                                                <button class="w-full text-left flex items-center gap-2 px-4 py-2 text-sm" :class="itemActive ? 'bg-gray-50 dark:bg-[#070a10]' : ''" @click="pruneImages(false)">
                                                    <Trash2Icon :size="13" /> {{ $t('pruneStandard') }}
                                                </button>
                                            </HMenuItem>
                                            <HMenuItem v-slot="{ active: itemActive }">
                                                <button class="w-full text-left flex items-center gap-2 px-4 py-2 text-sm" :class="itemActive ? 'bg-gray-50 dark:bg-[#070a10]' : ''" @click="pruneImages(true)">
                                                    <Trash2Icon :size="13" /> {{ $t('pruneAggressive') }}
                                                </button>
                                            </HMenuItem>
                                        </HMenuItems>
                                    </transition>
                                </HMenu>
                            </div>

                            <!-- Busy indicator -->
                            <div v-if="pruning" class="mt-2 text-sm text-gray-500 dark:text-gray-400 flex items-center justify-center gap-2">
                                <LoaderIcon :size="14" class="animate-spin" />
                                {{ $t('pruningImages') }}
                            </div>

                            <!-- Result -->
                            <div v-if="pruneResult !== null && !pruning" class="mt-2">
                                <span v-if="pruneResult.count > 0" class="text-green-500 flex items-center justify-center gap-1">
                                    <CheckCircleIcon :size="14" />
                                    {{ $t('pruneSuccess', [pruneResult.count, pruneResult.spaceReclaimed]) }}
                                </span>
                                <span v-else class="text-gray-400 flex items-center justify-center gap-1">
                                    <CheckCircleIcon :size="14" />
                                    {{ $t('pruneNothingFound') }}
                                </span>

                                <!-- Collapsible image list -->
                                <div v-if="pruneResult.count > 0" class="mt-2">
                                    <button class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 flex items-center gap-1 mx-auto" @click="pruneExpanded = !pruneExpanded">
                                        <ChevronDownIcon :size="12" :class="pruneExpanded ? 'rotate-180' : ''" class="transition-transform" />
                                        {{ pruneExpanded ? $t('pruneHideImages') : $t('pruneShowImages') }}
                                    </button>
                                    <div v-if="pruneExpanded" class="mt-2 text-left text-xs font-mono bg-gray-50 dark:bg-[#070a10] rounded-lg p-3 max-h-40 overflow-y-auto">
                                        <div v-for="img in pruneResult.images" :key="img" class="text-gray-600 dark:text-gray-400 py-0.5">
                                            {{ img }}
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>

                    <!-- Docker Run -->
                    <h2 class="mb-3">{{ $t("Docker Run") }}</h2>
                    <div class="mb-3">
                        <textarea id="name" v-model="dockerRunCommand" class="form-control docker-run shadow-box" required placeholder="docker run ..."></textarea>
                    </div>
                    <button class="btn btn-normal mb-4" @click="convertDockerRun">{{ $t("Convert to Compose") }}</button>
                </div>

                <!-- Right -->
                <div class="w-full md:w-[calc(42%-0.5rem)]">
                    <!-- Agent List -->
                    <div class="shadow-box big-padding">
                        <h4 class="mb-3">{{ $t("dockgeAgent", 2) }} <span class="badge bg-warning" style="font-size: 12px;">beta</span></h4>

                        <div v-for="(agentItem, ep) in $root.agentList" :key="ep" class="mb-3 agent flex items-center gap-2">
                            <template v-if="$root.agentStatusList[ep]">
                                <span v-if="$root.agentStatusList[ep] === 'online'" class="badge bg-primary">{{ $t("agentOnline") }}</span>
                                <span v-else-if="$root.agentStatusList[ep] === 'offline'" class="badge bg-danger">{{ $t("agentOffline") }}</span>
                                <span v-else class="badge bg-secondary">{{ $t($root.agentStatusList[ep]) }}</span>
                            </template>

                            <span v-if="ep === ''">{{ $t("currentEndpoint") }}</span>
                            <a v-else :href="agentItem.url" target="_blank">{{ ep }}</a>

                            <Trash2Icon v-if="ep !== ''" :size="14" class="ml-2 cursor-pointer text-white/30 hover:text-white/60" @click="removeAgentConfirm = agentItem.url" />
                        </div>

                        <button v-if="!showAgentForm" class="btn btn-normal" @click="showAgentForm = !showAgentForm">{{ $t("addAgent") }}</button>

                        <form v-if="showAgentForm" @submit.prevent="addAgent">
                            <div class="mb-3">
                                <label for="url" class="form-label">{{ $t("dockgeURL") }}</label>
                                <input id="url" v-model="agent.url" type="url" class="form-control" required placeholder="http://">
                            </div>
                            <div class="mb-3">
                                <label for="username" class="form-label">{{ $t("Username") }}</label>
                                <input id="username" v-model="agent.username" type="text" class="form-control" required>
                            </div>
                            <div class="mb-3">
                                <label for="password" class="form-label">{{ $t("Password") }}</label>
                                <input id="password" v-model="agent.password" type="password" class="form-control" required autocomplete="new-password">
                            </div>
                            <button type="submit" class="btn btn-primary" :disabled="connectingAgent">
                                <template v-if="connectingAgent">{{ $t("connecting") }}</template>
                                <template v-else>{{ $t("connect") }}</template>
                            </button>
                        </form>
                    </div>
                </div>
            </div>
        </div>
    </transition>

    <!-- Remove Agent Dialog -->
    <TransitionRoot appear :show="!!removeAgentConfirm" as="template">
        <HDialog as="div" class="relative z-50" @close="removeAgentConfirm = null">
            <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0" enter-to="opacity-100" leave="duration-150 ease-in" leave-from="opacity-100" leave-to="opacity-0">
                <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" />
            </TransitionChild>
            <div class="fixed inset-0 flex items-center justify-center p-4">
                <TransitionChild as="template" enter="duration-200 ease-out" enter-from="opacity-0 scale-95" enter-to="opacity-100 scale-100" leave="duration-150 ease-in" leave-from="opacity-100 scale-100" leave-to="opacity-0 scale-95">
                    <HDialogPanel class="modal-content w-full max-w-md bg-white dark:bg-[#0d1117] rounded-2xl shadow-2xl p-6">
                        <p class="mb-1 text-sm font-medium">{{ removeAgentConfirm }}</p>
                        <p class="mb-6 text-sm text-gray-700 dark:text-[#b1b8c0]">{{ $t("removeAgentMsg") }}</p>
                        <div class="flex justify-end gap-2">
                            <button class="btn btn-secondary" @click="removeAgentConfirm = null">{{ $t("cancel") }}</button>
                            <button class="btn btn-danger" @click="removeAgent(removeAgentConfirm)">{{ $t("removeAgent") }}</button>
                        </div>
                    </HDialogPanel>
                </TransitionChild>
            </div>
        </HDialog>
    </TransitionRoot>

    <router-view ref="child" />
</template>

<script>
import { statusNameShort } from "../../../common/util-common";
import { Dialog as HDialog, DialogPanel as HDialogPanel, TransitionRoot, TransitionChild, Menu as HMenu, MenuButton as HMenuButton, MenuItems as HMenuItems, MenuItem as HMenuItem } from "@headlessui/vue";
import { RefreshCwIcon, ArrowUpCircleIcon, CheckCircleIcon, Trash2Icon, ChevronDownIcon, LoaderIcon } from "lucide-vue-next";

export default {
    components: {
        HDialog,
        HDialogPanel,
        TransitionRoot,
        TransitionChild,
        HMenu,
        HMenuButton,
        HMenuItems,
        HMenuItem,
        RefreshCwIcon,
        ArrowUpCircleIcon,
        CheckCircleIcon,
        Trash2Icon,
        ChevronDownIcon,
        LoaderIcon,
    },
    props: {
        calculatedHeight: {
            type: Number,
            default: 0
        }
    },
    data() {
        return {
            dockerRunCommand: "",
            showAgentForm: false,
            removeAgentConfirm: null,
            connectingAgent: false,
            agent: {
                url: "http://",
                username: "",
                password: "",
            },
            checkingUpdates: false,
            updateCheckDone: false,
            updatesAvailableNum: 0,
            totalCheckedNum: 0,
            pruning: false,
            pruneResult: null,
            pruneExpanded: false,
        };
    },

    computed: {
        activeNum() {
            return this.getStatusNum("active");
        },
        inactiveNum() {
            return this.getStatusNum("inactive");
        },
        exitedNum() {
            return this.getStatusNum("exited");
        },
    },

    methods: {

        addAgent() {
            this.connectingAgent = true;
            this.$root.getSocket().emit("addAgent", this.agent, (res) => {
                this.$root.toastRes(res);

                if (res.ok) {
                    this.showAgentForm = false;
                    this.agent = {
                        url: "http://",
                        username: "",
                        password: "",
                    };
                }

                this.connectingAgent = false;
            });
        },

        removeAgent(url) {
            this.removeAgentConfirm = null;
            this.$root.getSocket().emit("removeAgent", url, (res) => {
                if (res.ok) {
                    this.$root.toastRes(res);

                    let urlObj = new URL(url);
                    let endpoint = urlObj.host;

                    // Remove the stack list and status list of the removed agent
                    delete this.$root.allAgentStackList[endpoint];
                }
            });
        },

        getStatusNum(statusName) {
            let num = 0;

            for (let stackName in this.$root.completeStackList) {
                const stack = this.$root.completeStackList[stackName];
                if (statusNameShort(stack.status) === statusName) {
                    num += 1;
                }
            }
            return num;
        },

        checkAllUpdates() {
            this.checkingUpdates = true;
            this.updateCheckDone = false;

            this.$root.emitAgent("", "checkAllStacksUpdates", (res) => {
                this.checkingUpdates = false;
                if (res.ok) {
                    this.updateCheckDone = true;
                    const results = res.allResults;
                    const stackNames = Object.keys(results);
                    this.totalCheckedNum = stackNames.length;
                    this.updatesAvailableNum = stackNames.filter(name => results[name].updateAvailable === true).length;
                } else {
                    this.$root.toastRes(res);
                }
            });
        },

        pruneImages(aggressive) {
            this.pruning = true;
            this.pruneResult = null;
            this.pruneExpanded = false;

            this.$root.emitAgent("", "pruneImages", aggressive, (res) => {
                this.pruning = false;
                if (res.ok) {
                    this.pruneResult = {
                        count: res.count,
                        spaceReclaimed: res.spaceReclaimed,
                        images: res.images,
                    };
                } else {
                    this.$root.toastRes(res);
                }
            });
        },

        convertDockerRun() {
            if (this.dockerRunCommand.trim() === "docker run") {
                throw new Error("Please enter a docker run command");
            }

            // composerize is working in dev, but after "vite build", it is not working
            // So pass to backend to do the conversion
            this.$root.getSocket().emit("composerize", this.dockerRunCommand, (res) => {
                if (res.ok) {
                    this.$root.composeTemplate = res.composeTemplate;
                    this.$router.push("/compose");
                } else {
                    this.$root.toastRes(res);
                }
            });
        },

    },
};
</script>

<style scoped>
.num {
    font-size: 30px;
    font-weight: bold;
    display: block;
}
.num.active { color: #74c2ff; }
.num.exited { color: #dc3545; }

.docker-run {
    border: none;
    font-family: 'JetBrains Mono', monospace;
    font-size: 15px;
}

.agent a { text-decoration: none; }
</style>
