import Vue from 'vue'
import Noty from 'noty'
import VueLazyload from 'vue-lazyload'
import i18n from '@/i18n'
import { disableExternal } from '@/utils/constants'

Vue.use(VueLazyload)

Vue.config.productionTip = true

const notyDefault = {
  type: 'info',
  layout: 'bottomRight',
  timeout: 3000,
  progressBar: true
}

Vue.prototype.$noty = (opts) => {
  new Noty(Object.assign({}, notyDefault, opts)).show()
}

Vue.prototype.$showSuccess = (message) => {
  new Noty(Object.assign({}, notyDefault, {
    text: message,
    type: 'success'
  })).show()
}

Vue.prototype.$showError = (error) => {
  let btns = [
    Noty.button(i18n.t('buttons.close'), '', function () {
      n.close()
    })
  ]

  if (!disableExternal) {
    btns.unshift(Noty.button(i18n.t('buttons.reportIssue'), '', function () {
      window.open('https://github.com/filebrowser/filebrowser/issues/new/choose')
    }))
  }

  let n = new Noty(Object.assign({}, notyDefault, {
    text: error.message || error,
    type: 'error',
    timeout: 3000,
    buttons: btns
  }))

  n.show()
}

Vue.directive('focus', {
  inserted: function (el) {
    el.focus()
  }
})

export default Vue
