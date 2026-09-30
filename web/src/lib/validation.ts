// 轻量表单校验：替代此前的 zod + react-hook-form 组合，
// 校验规则与后端 api.go 保持一致，错误文案不变。

// ---------- 表单数据类型 ----------

export interface LoginFormValues {
  username: string
  password: string
}

export interface DictFormValues {
  name: string
  color: string
  sort: number
  enabled: boolean
}

export type Errors<F> = Partial<Record<keyof F, string>>

// ---------- 校验规则 ----------

export function validateLogin(v: LoginFormValues): Errors<LoginFormValues> {
  const e: Errors<LoginFormValues> = {}
  if (!v.username) e.username = '请输入用户名'
  if (!v.password) e.password = '请输入密码'
  return e
}

// 教师表单：字段与文案与后端 api.go 的 validateTeacher 一一对应。
export interface TeacherFormValues {
  name: string
  gender: 'male' | 'female'
  age: string // 文本框输入，空串表示未填
  subject: string
  has_cert: boolean
  phone: string
  education: string
  university: string
  major: string
  remark: string
}

export function validateTeacher(v: TeacherFormValues): Errors<TeacherFormValues> {
  const e: Errors<TeacherFormValues> = {}
  const name = v.name.trim()
  if (!name) e.name = '姓名不能为空'
  else if ([...name].length > 50) e.name = '姓名过长（最多 50 个字符）'

  if (v.gender !== 'male' && v.gender !== 'female') e.gender = '性别只能是男或女'

  const age = v.age.trim()
  if (age !== '') {
    const n = Number(age)
    if (!Number.isInteger(n) || n < 18 || n > 100) e.age = '年龄须在 18-100 之间'
  }

  const phone = v.phone.trim()
  if (phone && !/^1[3-9]\d{9}$/.test(phone)) e.phone = '请输入正确的 11 位手机号'

  if ([...v.university.trim()].length > 100) e.university = '毕业院校过长（最多 100 个字符）'
  if ([...v.major.trim()].length > 100) e.major = '专业过长（最多 100 个字符）'
  if ([...v.remark.trim()].length > 500) e.remark = '备注过长（最多 500 个字符）'
  return e
}

// 字典表单（学科 / 学历）
export function validateDict(v: DictFormValues): Errors<DictFormValues> {
  const e: Errors<DictFormValues> = {}
  const name = v.name.trim()
  if (!name) e.name = '名称不能为空'
  else if ([...name].length > 32) e.name = '名称过长（最多 32 个字符）'
  if (!/^#[0-9a-fA-F]{6}$/.test(v.color)) e.color = '颜色格式须为 #RRGGBB'
  return e
}

export interface PasswordChangeValues {
  old_password: string
  new_password: string
  confirm: string
}

export function validatePasswordChange(v: PasswordChangeValues): Errors<PasswordChangeValues> {
  const e: Errors<PasswordChangeValues> = {}
  if (!v.old_password) e.old_password = '请输入旧密码'
  if (!v.new_password) e.new_password = '请输入新密码'
  else if (v.new_password.length < 6) e.new_password = '新密码长度须至少6位'
  if (!v.confirm) e.confirm = '请再次输入新密码'
  else if (v.confirm !== v.new_password) e.confirm = '两次输入不一致'
  return e
}