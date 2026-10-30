import { defineStore } from 'pinia'

let timer: ReturnType<typeof setTimeout> | undefined

export const useUiStore = defineStore('ui', {
  state: () => ({
    banner: '',
    pageTitle: '',
  }),
  actions: {
    /** 顶部绿色横幅，2 秒自动消失 */
    showBanner(msg: string) {
      this.banner = msg
      if (timer) clearTimeout(timer)
      timer = setTimeout(() => {
        this.banner = ''
      }, 2000)
    },
    setTitle(title: string) {
      this.pageTitle = title
    },
  },
})
