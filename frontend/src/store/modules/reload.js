import Vue from 'vue'
import { reload as api } from '@/api'
import buttons from '@/utils/buttons'
import url from '@/utils/url'

const state = {
    id: 0,
    progress: [],
    queue: [],
    reloads: {}
}

const mutations = {
    setProgress(state, id) {
        Vue.set(state.progress, id)
    },
    reset: (state) => {
        state.id = 0
        state.progress = []
    },
    addJob: (state, item) => {
        state.queue.push(item)
        state.id++
    },
    moveJob(state) {
        const item = state.queue[0]
        state.queue.shift()
        Vue.set(state.reloads, item.id, item)
    },
    removeJob(state, id) {
        delete state.reloads[id]
    }
}

const beforeUnload = (event) => {
    event.preventDefault()
    event.returnValue = ''
}

const actions = {
    reload: (context, item) => {
        let reloadsCount = Object.keys(context.state.reloads).length;

        let isQueueEmpty = context.state.queue.length == 0
        let isReloadsEmpty = reloadsCount == 0

        if (isQueueEmpty && isReloadsEmpty) {
            window.addEventListener('beforeunload', beforeUnload)
            buttons.loading('reload')
        }

        context.commit('addJob', item)
        context.dispatch('processReloads')
    },
    finishReload: (context, item) => {
        context.commit('setProgress', item.id)
        context.commit('removeJob', item.id)
        context.dispatch('processReloads')
    },
    processReloads: async (context) => {
        let reloadsCount = Object.keys(context.state.reloads).length;

        let isQueueEmpty = context.state.queue.length == 0
        let isReloadsEmpty = reloadsCount == 0

        let isFinished = isQueueEmpty && isReloadsEmpty
        let canProcess = !isQueueEmpty && isReloadsEmpty

        if (isFinished) {
            window.removeEventListener('beforeunload', beforeUnload)
            buttons.success('reload')
            context.commit('reset')
            context.commit('setReload', true, { root: true })
        }

        if (canProcess) {
            const item = context.state.queue[0];
            context.commit('moveJob')
            let res = await api.reload().catch(Vue.prototype.$showError)
            let msg = url.unicodeToChar(await res.text())

            if (res.status == 200) {
                Vue.prototype.$showSuccess("reload config success")
            } else {
                Vue.prototype.$showError("reload config failed: " + msg)
            }
            context.dispatch('finishReload', item)
        }
    }
}

export default { state, mutations, actions, namespaced: true }