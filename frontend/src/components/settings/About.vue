<template>
    <div class="flex justify-center items-center">
        <div class="flex flex-col justify-center items-center my-16 mx-4">
            <object class="my-4" width="200" height="200" data="/icon.svg" />
            <div class="text-xl font-bold">Dockge</div>
            <div>{{ $t("Version") }}: {{ $root.info.version }}</div>
            <div class="text-sm text-[#cccccc] dark:text-[#333333]">{{ $t("Frontend Version") }}: {{ $root.frontendVersion }}</div>

            <div v-if="!$root.isFrontendBackendVersionMatched" class="alert alert-warning mt-4" role="alert">
                ⚠️ {{ $t("Frontend Version do not match backend version!") }}
            </div>
        </div>
    </div>
</template>

<script>
export default {};
</script>
