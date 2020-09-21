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
    let url = "/api/reload"
    let uuid = store.state.uuid
    if (uuid != undefined && uuid != "") {
        url += `?uuid=${uuid}`
    }
    const res = await fetchURL(url, opts)
    // reset uuid after reload
    store.commit('setUUID', '')
    return res
}
