<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { NModal, NInput, NInputNumber, NDatePicker, NButton } from 'naive-ui'
import api from '@/api'
import type { Pet } from '@/types'
import { useUiStore } from '@/stores/ui'

const props = defineProps<{ show: boolean; pet: Pet | null }>()
const emit = defineEmits<{
  (e: 'update:show', v: boolean): void
  (e: 'saved', pet: Pet): void
}>()

const ui = useUiStore()
const isEdit = computed(() => !!props.pet)

const form = ref({
  name: '',
  breed: '',
  gender: '' as '' | 'male' | 'female',
  birth_date: '',
  weight: null as number | null,
  notes: '',
})
const nameError = ref('')
const avatarError = ref('')
const formError = ref('')
const avatarFile = ref<File | null>(null)
const avatarPreview = ref('')
const fileInput = ref<HTMLInputElement | null>(null)
const submitting = ref(false)

watch(
  () => props.show,
  (v) => {
    if (!v) return
    nameError.value = ''
    avatarError.value = ''
    formError.value = ''
    avatarFile.value = null
    if (props.pet) {
      form.value = {
        name: props.pet.name,
        breed: props.pet.breed || '',
        gender: (props.pet.gender || '') as '' | 'male' | 'female',
        birth_date: props.pet.birth_date || '',
        weight: props.pet.weight ?? null,
        notes: props.pet.notes || '',
      }
      avatarPreview.value = props.pet.avatar_url || ''
    } else {
      form.value = { name: '', breed: '', gender: '', birth_date: '', weight: null, notes: '' }
      avatarPreview.value = ''
    }
  },
)

function validateName() {
  nameError.value = form.value.name.trim() ? '' : '请输入名字'
}

function onPickAvatar() {
  fileInput.value?.click()
}

function onFileChange(e: Event) {
  const input = e.target as HTMLInputElement
  const f = input.files?.[0]
  if (!f) return
  if (f.size > 2 * 1024 * 1024) {
    avatarError.value = '头像不能超过 2MB'
    input.value = ''
    return
  }
  avatarError.value = ''
  avatarFile.value = f
  avatarPreview.value = URL.createObjectURL(f)
  input.value = ''
}

async function uploadAvatar(petId: string) {
  if (!avatarFile.value) return
  const fd = new FormData()
  fd.append('file', avatarFile.value)
  await api.post(`/pets/${petId}/avatar`, fd)
}

async function submit() {
  validateName()
  if (nameError.value || avatarError.value || submitting.value) return
  submitting.value = true
  formError.value = ''
  try {
    const body = {
      name: form.value.name.trim(),
      breed: form.value.breed.trim(),
      gender: form.value.gender,
      birth_date: form.value.birth_date || '',
      weight: form.value.weight ?? 0,
      notes: form.value.notes.trim(),
    }
    let pet: Pet
    if (isEdit.value && props.pet) {
      const { data } = await api.put(`/pets/${props.pet.id}`, body)
      pet = data
    } else {
      const { data } = await api.post('/pets', body)
      pet = data
    }
    await uploadAvatar(pet.id)
    emit('update:show', false)
    emit('saved', pet)
  } catch (err) {
    formError.value = (err as Error).message
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <n-modal
    :show="show"
    @update:show="emit('update:show', $event)"
    preset="card"
    :title="isEdit ? '编辑宠物' : '添加宠物'"
    :style="{ width: 'calc(100vw - 32px)', maxWidth: '480px', borderRadius: '12px' }"
    :bordered="false"
  >
    <div class="field-item avatar-field">
      <div class="avatar-upload" role="button" @click="onPickAvatar">
        <img v-if="avatarPreview" :src="avatarPreview" class="avatar avatar-72" alt="头像" />
        <div v-else class="avatar-fallback avatar-72 avatar-empty">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="3" width="18" height="18" rx="4" />
            <circle cx="8.5" cy="8.5" r="1.5" />
            <polyline points="21 15 16 10 5 21" />
          </svg>
        </div>
      </div>
      <div class="avatar-tip">点击上传头像（不超过 2MB）</div>
      <div v-if="avatarError" class="field-error">{{ avatarError }}</div>
      <input ref="fileInput" type="file" accept="image/*" hidden @change="onFileChange" />
    </div>

    <div class="field-item">
      <label class="field-label">名字 <span class="required">*</span></label>
      <n-input v-model:value="form.name" placeholder="请输入名字" :maxlength="20" @blur="validateName" />
      <div v-if="nameError" class="field-error">{{ nameError }}</div>
    </div>

    <div class="field-item">
      <label class="field-label">品种</label>
      <n-input v-model:value="form.breed" placeholder="如：金毛、英短" :maxlength="30" />
    </div>

    <div class="field-item">
      <label class="field-label">性别</label>
      <div class="chip-group">
        <button type="button" class="chip" :class="{ 'chip-active': form.gender === 'male' }" @click="form.gender = 'male'">公</button>
        <button type="button" class="chip" :class="{ 'chip-active': form.gender === 'female' }" @click="form.gender = 'female'">母</button>
      </div>
    </div>

    <div class="field-item">
      <label class="field-label">出生日期</label>
      <n-date-picker
        v-model:formatted-value="form.birth_date"
        type="date"
        value-format="yyyy-MM-dd"
        clearable
        placeholder="选择出生日期"
        style="width: 100%"
      />
    </div>

    <div class="field-item">
      <label class="field-label">体重（kg）</label>
      <n-input-number v-model:value="form.weight" :min="0" :max="200" placeholder="如：5.2" style="width: 100%" />
    </div>

    <div class="field-item">
      <label class="field-label">备注</label>
      <n-input v-model:value="form.notes" type="textarea" :rows="3" placeholder="性格、喜好、注意事项等" :maxlength="200" />
    </div>

    <div v-if="formError" class="field-error form-error">{{ formError }}</div>

    <n-button type="primary" block :loading="submitting" class="submit-btn" @click="submit">保存</n-button>
  </n-modal>
</template>

<style scoped>
.avatar-field {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.avatar-upload {
  cursor: pointer;
}

.avatar-72 {
  width: 72px;
  height: 72px;
  font-size: 24px;
}

.avatar-empty {
  color: var(--text-sub);
}

.avatar-tip {
  font-size: 12px;
  color: var(--text-sub);
  margin-top: 6px;
}

.required {
  color: var(--danger);
}

.form-error {
  margin-top: 12px;
}

.submit-btn {
  margin-top: 20px;
}
</style>
