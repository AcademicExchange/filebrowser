import { fetchURL } from './utils'
import store from '@/store'

export async function reload() {
    return reloadAction('GET')
}

async function reloadAction(method, content) {
    let opts = { method }
    if (content) {
        opts.body = content
    }
    let url = `/api/reload?uuid=${store.state.uuid}`
    // reset uuid before reload
    store.commit('setUUID', '')
    const res = await fetchURL(url, opts)
    return res
}
