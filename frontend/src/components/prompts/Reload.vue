<template>
<div class="card floating">
    <div class="card-title">
        <h2>{{ $t('prompts.reload') }}</h2>
    </div>

    <div class="card-content">
        <p>{{ $t('prompts.reloadMessage') }}</p>
    </div>

    <div class="card-action full">
        <button class="button button--flat button--grey" 
            @click="$store.commit('closeHovers')" 
            :aria-label="$t('buttons.cancel')" 
            :title="$t('buttons.cancel')">{{ $t('buttons.cancel') }}
        </button>
        <button class="button button--flat button--blue" 
            @click="confirmReload" 
            :aria-label="$t('buttons.reload')" 
            :title="$t('buttons.reload')">{{ $t('buttons.reload') }}
        </button>
    </div>
</div>
</template>

<script>

export default {
    name: "reload",
    methods: {
        confirmReload: async function () {
            this.$store.commit("closeHovers")
            let id = this.$store.state.reloads.id
            const item = { id }
            try {
                this.$store.dispatch('reloads/reload', item);
            } catch (e) {
                this.$showError(e)
            }
        }
    }
}
</script>
