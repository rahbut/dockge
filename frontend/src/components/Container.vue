<template>
    <div class="shadow-box big-padding mb-3">
        <div class="flex gap-2">
            <div class="flex-1 min-w-0">
                <h4>{{ name }}</h4>
                <div class="image mb-2 text-sm text-gray-500 dark:text-gray-300">
                    <span class="mr-1">{{ imageName }}:</span><span class="tag">{{ imageTag }}</span>
                </div>
                <div v-if="!isEditMode" class="flex flex-wrap gap-1">
                    <span class="badge" :class="bgStyle">{{ status }}</span>

                    <span v-if="updateStatus && updateStatus.updateAvailable" class="badge bg-warning" :title="$t('updateAvailable')">
                        <ArrowUpCircleIcon :size="12" class="mr-1 inline" />{{ $t('updateAvailable') }}
                    </span>
                    <span v-else-if="updateStatus && updateStatus.error === 'registryError'" class="badge bg-secondary" :title="updateStatus.error">
                        {{ $t(updateStatus.error) || updateStatus.error }}
                    </span>

                    <a v-for="port in (ports ?? envsubstService.ports)" :key="port" :href="parsePort(port).url" target="_blank">
                        <span class="badge bg-secondary">{{ parsePort(port).display }}</span>
                    </a>
                </div>
            </div>
            <div class="flex items-center justify-end shrink-0">
                <router-link v-if="!isEditMode" class="btn btn-normal" :to="terminalRouteLink">
                    <TerminalIcon :size="14" class="mr-1" /> Bash
                </router-link>
            </div>
        </div>

        <div v-if="isEditMode" class="mt-2 flex gap-2">
            <button class="btn btn-normal" @click="showConfig = !showConfig">
                <PencilIcon :size="14" class="mr-1" /> {{ $t("Edit") }}
            </button>
            <button class="btn btn-danger" @click="remove">
                <Trash2Icon :size="14" class="mr-1" /> {{ $t("deleteContainer") }}
            </button>
        </div>

        <transition name="slide-fade" appear>
            <div v-if="isEditMode && showConfig" class="config mt-3">
                <!-- Image -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("dockerImage") }}
                    </label>
                    <div class="input-group mb-3">
                        <input
                            v-model="service.image"
                            class="form-control"
                            list="image-datalist"
                        />
                    </div>

                    <!-- TODO: Search online: https://hub.docker.com/api/content/v1/products/search?q=louislam%2Fuptime&source=community&page=1&page_size=4 -->
                    <datalist id="image-datalist">
                        <option value="louislam/uptime-kuma:1" />
                    </datalist>
                    <div class="form-text"></div>
                </div>

                <!-- Ports -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("port", 2) }}
                    </label>
                    <ArrayInput name="ports" :display-name="$t('port')" placeholder="HOST:CONTAINER" />
                </div>

                <!-- Volumes -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("volume", 2) }}
                    </label>
                    <ArrayInput name="volumes" :display-name="$t('volume')" placeholder="HOST:CONTAINER" />
                </div>

                <!-- Restart Policy -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("restartPolicy") }}
                    </label>
                    <select v-model="service.restart" class="form-select">
                        <option value="always">{{ $t("restartPolicyAlways") }}</option>
                        <option value="unless-stopped">{{ $t("restartPolicyUnlessStopped") }}</option>
                        <option value="on-failure">{{ $t("restartPolicyOnFailure") }}</option>
                        <option value="no">{{ $t("restartPolicyNo") }}</option>
                    </select>
                </div>

                <!-- Environment Variables -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("environmentVariable", 2) }}
                    </label>
                    <ArrayInput name="environment" :display-name="$t('environmentVariable')" placeholder="KEY=VALUE" />
                </div>

                <!-- Network -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("network", 2) }}
                    </label>

                    <div v-if="networkList.length === 0 && service.networks && service.networks.length > 0" class="text-yellow-500 mb-3">
                        {{ $t("NoNetworksAvailable") }}
                    </div>

                    <ArraySelect name="networks" :display-name="$t('network')" placeholder="Network Name" :options="networkList" />
                </div>

                <!-- Depends on -->
                <div class="mb-4">
                    <label class="form-label">
                        {{ $t("dependsOn") }}
                    </label>
                    <ArrayInput name="depends_on" :display-name="$t('dependsOn')" :placeholder="$t(`containerName`)" />
                </div>
            </div>
        </transition>
    </div>
</template>

<script>
import { defineComponent } from "vue";
import { parseDockerPort } from "../../../common/util-common";
import { ArrowUpCircleIcon, TerminalIcon, PencilIcon, Trash2Icon } from "lucide-vue-next";
import ArrayInput from "./ArrayInput.vue";
import ArraySelect from "./ArraySelect.vue";

export default defineComponent({
    components: {
        ArrowUpCircleIcon,
        TerminalIcon,
        PencilIcon,
        Trash2Icon,
        ArrayInput,
        ArraySelect,
    },
    props: {
        name: {
            type: String,
            required: true,
        },
        isEditMode: {
            type: Boolean,
            default: false,
        },
        first: {
            type: Boolean,
            default: false,
        },
        status: {
            type: String,
            default: "N/A",
        },
        ports: {
            type: Array,
            default: null
        },
        updateStatus: {
            type: Object,
            default: null
        }
    },
    emits: [
    ],
    data() {
        return {
            showConfig: false,
        };
    },
    computed: {

        networkList() {
            let list = [];
            for (const networkName in this.jsonObject.networks) {
                list.push(networkName);
            }
            return list;
        },

        bgStyle() {
            if (this.status === "running" || this.status === "healthy") {
                return "bg-primary";
            } else if (this.status === "unhealthy") {
                return "bg-danger";
            } else {
                return "bg-secondary";
            }
        },

        terminalRouteLink() {
            if (this.endpoint) {
                return {
                    name: "containerTerminalEndpoint",
                    params: {
                        endpoint: this.endpoint,
                        stackName: this.stackName,
                        serviceName: this.name,
                        type: "bash",
                    },
                };
            } else {
                return {
                    name: "containerTerminal",
                    params: {
                        stackName: this.stackName,
                        serviceName: this.name,
                        type: "bash",
                    },
                };
            }
        },

        endpoint() {
            return this.$parent.$parent.endpoint;
        },

        stack() {
            return this.$parent.$parent.stack;
        },

        stackName() {
            return this.$parent.$parent.stack.name;
        },

        service() {
            if (!this.jsonObject.services[this.name]) {
                return {};
            }
            return this.jsonObject.services[this.name];
        },

        jsonObject() {
            return this.$parent.$parent.jsonConfig;
        },

        envsubstJSONConfig() {
            return this.$parent.$parent.envsubstJSONConfig;
        },

        envsubstService() {
            if (!this.envsubstJSONConfig.services[this.name]) {
                return {};
            }
            return this.envsubstJSONConfig.services[this.name];
        },

        imageName() {
            if (this.envsubstService.image) {
                return this.envsubstService.image.split(":")[0];
            } else {
                return "";
            }
        },

        imageTag() {
            if (this.envsubstService.image) {
                let tag = this.envsubstService.image.split(":")[1];

                if (tag) {
                    return tag;
                } else {
                    return "latest";
                }
            } else {
                return "";
            }
        },
    },
    methods: {
        parsePort(port) {
            if (this.stack.endpoint) {
                return parseDockerPort(port, this.stack.primaryHostname);
            } else {
                let hostname = this.$root.info.primaryHostname || location.hostname;
                return parseDockerPort(port, hostname);
            }
        },
        remove() {
            delete this.jsonObject.services[this.name];
        },
    }
});
</script>

<style scoped>
.image { font-size: 0.8rem; color: #6c757d; }
.image .tag { color: #33383b; }
.dark .image { color: #9ca3af; }
.dark .image .tag { color: #d1d5db; }
</style>
