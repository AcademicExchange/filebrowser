<template>
  <button :title="$t('buttons.reload')" :aria-label="$t('buttons.reload')" class="action" @click="show" id="reload-button">
    <i class="material-icons">sync</i>
    <span>{{ $t('buttons.reload') }}</span>
  </button>
</template>

<script>
export default {
    name: 'reload-button',
    methods: {
        show: function () {
            this.$store.commit('showHover', 'reload')
        }
    }
}
</script>
