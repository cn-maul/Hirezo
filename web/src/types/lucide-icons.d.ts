// lucide 一律用深路径导入（绕开 barrel，dev 预打包从 ~1800 模块降到 ~250）；
// 深路径无类型声明，此处统一补齐。注意 0.475 中部分图标已改名：
// CheckCircle2→circle-check、AlertCircle→triangle-alert、UploadCloud→cloud-upload
declare module 'lucide-vue-next/dist/esm/icons/*.js' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, {}, any>
  export default component
}
