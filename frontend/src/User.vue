<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import Alert from 'ant-design-vue/es/alert';
import AntApp from 'ant-design-vue/es/app';
import Avatar from 'ant-design-vue/es/avatar';
import Button from 'ant-design-vue/es/button';
import Card from 'ant-design-vue/es/card';
import Col from 'ant-design-vue/es/col';
import ConfigProvider from 'ant-design-vue/es/config-provider';
import Empty from 'ant-design-vue/es/empty';
import Form from 'ant-design-vue/es/form';
import Input from 'ant-design-vue/es/input';
import Modal from 'ant-design-vue/es/modal';
import Row from 'ant-design-vue/es/row';
import Select from 'ant-design-vue/es/select';
import Space from 'ant-design-vue/es/space';
import Tag from 'ant-design-vue/es/tag';
import Typography from 'ant-design-vue/es/typography';
import message from 'ant-design-vue/es/message';
import zhCN from 'ant-design-vue/es/locale/zh_CN';
import { Link, LogOut, QrCode } from 'lucide-vue-next';

type ApiEnvelope<T> = {
  status: boolean;
  message: string;
  data: T;
};

type PublicUser = {
  id: string;
  username: string;
  nickname: string;
  created_at: number;
};

type Bindings = {
  qq?: string;
  telegram?: string;
  yybgo_openid?: string;
  yybgo_openids?: string[];
  updated_at?: number;
};

type YybGoPanel = {
  index: number;
  id: string;
  name: string;
  status: string;
  message: string;
};

const tokenKey = 'shaniu_user_token';
const token = ref(localStorage.getItem(tokenKey) || '');
const loading = ref(true);
const user = ref<PublicUser | null>(null);
const bindings = reactive<Bindings>({});
const panels = ref<YybGoPanel[]>([]);
const selectedPanel = ref(1);
const bindForm = reactive({
  qq: '',
  telegram: '',
});
const yybgo = reactive({
  qrType: 1,
  uuid: '',
  qrOpen: false,
  qrLoading: false,
  confirmLoading: false,
  qrResult: null as unknown,
});

const userInitial = computed(() => {
  const name = user.value?.nickname || user.value?.username || 'U';
  return name.slice(0, 1).toUpperCase();
});

const panelOptions = computed(() => panels.value.map((item) => ({
  value: item.index,
  label: `编号 ${item.index}`,
})));
const qrTypeOptions = [
  { value: 1, label: '应用宝' },
  { value: 2, label: '手游助手' },
];

const selectedPanelText = computed(() => `编号 ${selectedPanel.value || 1}`);
const yybgoOpenids = computed(() => normalizeOpenids(bindings));

const qrImage = computed(() => {
  const value = findValueByKey(yybgo.qrResult, [
    'qrcodeUrl',
    'qrCodeDataUrl',
    'qrcode',
    'qrCode',
    'qr_code',
    'image',
    'url',
    'qrUrl',
    'qr_url',
  ]);
  if (typeof value !== 'string') return '';
  if (/^(https?:\/\/|data:image\/)/i.test(value)) return value;
  return '';
});

async function requestJSON<T>(url: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json');
  }
  if (token.value && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token.value}`);
  }
  const res = await fetch(url, {
    credentials: 'include',
    ...options,
    headers,
  });
  const payload = (await res.json().catch(() => ({
    status: false,
    message: '服务响应异常',
    data: null,
  }))) as ApiEnvelope<T>;
  if (!res.ok || payload.status === false) {
    throw new Error(payload.message || '请求失败');
  }
  return payload.data;
}

function fillProfile(data: { user: PublicUser; bindings: Bindings; yybgo_panels: YybGoPanel[] }) {
  user.value = data.user;
  Object.assign(bindings, data.bindings || {});
  bindForm.qq = bindings.qq || '';
  bindForm.telegram = bindings.telegram || '';
  panels.value = data.yybgo_panels || [];
  selectedPanel.value = panels.value[0]?.index || 1;
}

async function loadProfile() {
  loading.value = true;
  try {
    const data = await requestJSON<{ user: PublicUser; bindings: Bindings; yybgo_panels: YybGoPanel[] }>('/api/user/profile');
    fillProfile(data);
  } catch (error) {
    localStorage.removeItem(tokenKey);
    token.value = '';
    user.value = null;
    message.error(error instanceof Error ? error.message : '请先登录');
  } finally {
    loading.value = false;
  }
}

async function logout() {
  try {
    await requestJSON<null>('/api/user/outlogin', {
      method: 'POST',
      body: '{}',
    });
  } catch (_) {
  } finally {
    localStorage.removeItem(tokenKey);
    window.location.href = '/';
  }
}

async function saveBinding(platform: 'qq' | 'telegram') {
  const value = platform === 'qq' ? bindForm.qq : bindForm.telegram;
  const data = await requestJSON<Bindings>('/api/user/bind', {
    method: 'PUT',
    body: JSON.stringify({ platform, value }),
  });
  Object.assign(bindings, data);
  message.success('绑定已保存');
}

async function removeBinding(platform: 'qq' | 'telegram') {
  const data = await requestJSON<Bindings>('/api/user/bind', {
    method: 'DELETE',
    body: JSON.stringify({ platform }),
  });
  Object.assign(bindings, data);
  if (platform === 'qq') bindForm.qq = '';
  if (platform === 'telegram') bindForm.telegram = '';
  message.success('绑定已解除');
}

async function openYybGoLogin() {
  if (!panels.value.length) {
    message.error('后台还没有绑定 yybgo');
    return;
  }
  yybgo.qrOpen = true;
  yybgo.qrResult = null;
  yybgo.uuid = '';
  yybgo.qrLoading = true;
  try {
    const data = await requestJSON<unknown>('/api/user/yybgo/qr/start', {
      method: 'POST',
      body: JSON.stringify({ panel: selectedPanel.value, type: yybgo.qrType }),
    });
    yybgo.qrResult = data;
    const uuid = findValueByKey(data, ['uuid', 'qrUuid', 'qr_uuid']);
    if (typeof uuid === 'string') yybgo.uuid = uuid;
    if (!yybgo.uuid) {
      message.warning('二维码已生成，但未识别到 uuid');
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '生成二维码失败');
    yybgo.qrOpen = false;
  } finally {
    yybgo.qrLoading = false;
  }
}

async function confirmYybGoLogin() {
  if (!yybgo.uuid.trim()) {
    message.error('缺少二维码 uuid，请重新生成二维码');
    return;
  }
  yybgo.confirmLoading = true;
  try {
    const data = await requestJSON<{ openid: string; bindings: Bindings }>('/api/user/yybgo/login/confirm', {
      method: 'POST',
      body: JSON.stringify({ panel: selectedPanel.value, uuid: yybgo.uuid.trim() }),
    });
    Object.assign(bindings, data.bindings || { yybgo_openid: data.openid, yybgo_openids: data.openid ? [data.openid] : [] });
    yybgo.qrOpen = false;
    message.success('yybgo 登录成功');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '未检测到 yybgo 登录');
  } finally {
    yybgo.confirmLoading = false;
  }
}

function findValueByKey(value: unknown, keys: string[]): unknown {
  if (!value || typeof value !== 'object') return undefined;
  const record = value as Record<string, unknown>;
  for (const key of keys) {
    if (record[key] !== undefined && record[key] !== null && record[key] !== '') {
      return record[key];
    }
  }
  for (const item of Object.values(record)) {
    const found = findValueByKey(item, keys);
    if (found !== undefined && found !== null && found !== '') return found;
  }
  return undefined;
}

function normalizeOpenids(value: Bindings) {
  const rows = [] as string[];
  if (value.yybgo_openid) rows.push(value.yybgo_openid);
  for (const item of value.yybgo_openids || []) {
    if (item) rows.push(item);
  }
  return Array.from(new Set(rows.map((item) => item.trim()).filter(Boolean)));
}

onMounted(loadProfile);
</script>

<template>
  <ConfigProvider :locale="zhCN">
    <AntApp>
      <div class="user-page">
        <header class="user-topbar">
          <a class="user-brand" href="/">
            <span class="user-brand-mark">S</span>
            <span>Shaniu</span>
          </a>
          <Space v-if="user" align="center">
            <Avatar :size="34" class="user-avatar">{{ userInitial }}</Avatar>
            <span class="user-name">{{ user.nickname || user.username }}</span>
            <Button @click="logout"><template #icon><LogOut :size="16" /></template>退出</Button>
          </Space>
        </header>

        <main class="user-content">
          <Card v-if="!loading && !user" class="user-login-card" :bordered="false">
            <Empty description="请先登录普通用户账号" />
            <Button type="primary" href="/">返回登录</Button>
          </Card>

          <template v-if="user">
            <section class="user-summary">
              <Card :bordered="false">
                <Space align="center">
                  <Avatar :size="56" class="user-avatar">{{ userInitial }}</Avatar>
                  <span>
                    <Typography.Title :level="3" class="user-title">{{ user.nickname || user.username }}</Typography.Title>
                    <Typography.Text class="muted">@{{ user.username }}</Typography.Text>
                  </span>
                </Space>
              </Card>
              <Card :bordered="false">
                <Space direction="vertical" size="small">
                  <Typography.Text strong>绑定状态</Typography.Text>
                  <Space wrap>
                    <Tag :color="bindings.qq ? 'green' : 'default'">QQ {{ bindings.qq || '未绑定' }}</Tag>
                    <Tag :color="bindings.telegram ? 'green' : 'default'">TG {{ bindings.telegram || '未绑定' }}</Tag>
                    <Tag :color="yybgoOpenids.length ? 'green' : 'default'">yybgo {{ yybgoOpenids.length ? `${yybgoOpenids.length} 个账号` : '未登录' }}</Tag>
                  </Space>
                </Space>
              </Card>
            </section>

            <Row :gutter="[16, 16]">
              <Col :xs="24" :lg="15">
                <Card class="user-panel" :bordered="false">
                  <template #title>
                    <Space><QrCode :size="18" />yybgo 账号</Space>
                  </template>

                  <Alert
                    v-if="!panels.length"
                    type="warning"
                    show-icon
                    message="后台还没有绑定 yybgo"
                    description="请管理员先在后台添加 yybgo 面板后，普通用户才能添加 yybgo 账号。"
                  />

                  <div v-else class="yybgo-account">
                    <div class="yybgo-status">
                      <Typography.Text strong>当前 yybgo openid</Typography.Text>
                      <Space v-if="yybgoOpenids.length" direction="vertical" size="small">
                        <Typography.Text v-for="openid in yybgoOpenids" :key="openid" class="mono">{{ openid }}</Typography.Text>
                      </Space>
                      <Typography.Text v-else class="muted">未登录</Typography.Text>
                    </div>
                    <Form layout="vertical" class="yybgo-form">
                      <Form.Item label="yybgo 面板">
                        <Select v-model:value="selectedPanel" :options="panelOptions" />
                      </Form.Item>
                      <Form.Item label="二维码类型">
                        <Select v-model:value="yybgo.qrType" :options="qrTypeOptions" />
                      </Form.Item>
                      <Button type="primary" size="large" @click="openYybGoLogin">
                        添加 yybgo 账号
                      </Button>
                    </Form>
                  </div>
                </Card>
              </Col>

              <Col :xs="24" :lg="9">
                <Card class="user-panel" :bordered="false">
                  <template #title>
                    <Space><Link :size="18" />账号绑定</Space>
                  </template>
                  <Form layout="vertical">
                    <template v-if="bindings.qq">
                      <Form.Item label="QQ 号">
                        <Space class="bound-row">
                          <Typography.Text class="mono">{{ bindings.qq }}</Typography.Text>
                          <Button @click="removeBinding('qq')">解绑</Button>
                        </Space>
                      </Form.Item>
                    </template>
                    <template v-else>
                      <Form.Item label="QQ 号">
                        <Input v-model:value="bindForm.qq" placeholder="例如：860562056" />
                      </Form.Item>
                      <Space class="bind-actions">
                        <Button type="primary" @click="saveBinding('qq')">绑定 QQ</Button>
                      </Space>
                    </template>

                    <template v-if="bindings.telegram">
                      <Form.Item label="Telegram ID" class="bind-field">
                        <Space class="bound-row">
                          <Typography.Text class="mono">{{ bindings.telegram }}</Typography.Text>
                          <Button @click="removeBinding('telegram')">解绑</Button>
                        </Space>
                      </Form.Item>
                    </template>
                    <template v-else>
                      <Form.Item label="Telegram ID" class="bind-field">
                        <Input v-model:value="bindForm.telegram" placeholder="例如：123456789" />
                      </Form.Item>
                      <Space class="bind-actions">
                        <Button type="primary" @click="saveBinding('telegram')">绑定 TG</Button>
                      </Space>
                    </template>
                  </Form>
                </Card>
              </Col>
            </Row>
          </template>
        </main>

        <Modal
          v-model:open="yybgo.qrOpen"
          title="添加 yybgo 账号"
          ok-text="确认登录"
          cancel-text="取消"
          :confirm-loading="yybgo.confirmLoading"
          @ok="confirmYybGoLogin"
        >
          <Space direction="vertical" size="middle" class="qr-modal">
            <Alert type="info" show-icon :message="`请在 2 分钟内使用 ${selectedPanelText} 扫码登录，完成后点击确认登录。`" />
            <div class="qr-box">
              <span v-if="yybgo.qrLoading" class="muted">二维码生成中...</span>
              <img v-else-if="qrImage" :src="qrImage" alt="yybgo 二维码" />
              <span v-else class="muted">未识别到二维码图片，请检查 yybgo 返回。</span>
            </div>
            <Typography.Text v-if="yybgo.uuid" class="mono">UUID: {{ yybgo.uuid }}</Typography.Text>
          </Space>
        </Modal>
      </div>
    </AntApp>
  </ConfigProvider>
</template>

<style scoped>
.user-page {
  min-height: 100vh;
  background: #f5f7fb;
}

.user-topbar {
  position: sticky;
  top: 0;
  z-index: 10;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 56px;
  padding: 0 20px;
  background: #ffffff;
  border-bottom: 1px solid #edf0f5;
}

.user-brand {
  display: flex;
  align-items: center;
  gap: 10px;
  color: #1f2937;
  font-weight: 700;
}

.user-brand-mark,
.user-avatar {
  background: #111827;
  color: #ffffff;
}

.user-brand-mark {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
}

.user-content {
  width: min(1180px, calc(100% - 32px));
  margin: 0 auto;
  padding: 18px 0 36px;
}

.user-summary {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 420px;
  gap: 16px;
  margin-bottom: 16px;
}

.user-panel,
.user-summary :deep(.ant-card),
.user-login-card {
  border: 1px solid #edf0f5;
  border-radius: 8px;
}

.user-login-card {
  max-width: 420px;
  margin: 80px auto 0;
  text-align: center;
}

.user-title {
  margin: 0 !important;
}

.user-name {
  font-weight: 700;
}

.yybgo-account {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 280px;
  gap: 16px;
  align-items: start;
}

.yybgo-status {
  display: grid;
  gap: 10px;
  min-height: 136px;
  padding: 16px;
  border: 1px solid #edf0f5;
  border-radius: 8px;
  background: #f8fafc;
}

.yybgo-form {
  padding: 16px;
  border: 1px solid #edf0f5;
  border-radius: 8px;
  background: #ffffff;
}

.qr-modal {
  width: 100%;
}

.qr-box {
  display: grid;
  place-items: center;
  min-height: 260px;
  padding: 16px;
  border: 1px solid #edf0f5;
  border-radius: 8px;
  background: #ffffff;
}

.qr-box img {
  max-width: 240px;
  width: 100%;
  height: auto;
}

.bind-actions {
  margin-bottom: 14px;
}

.bound-row {
  width: 100%;
  justify-content: space-between;
}

.bind-field {
  margin-top: 12px;
}

.muted {
  color: #6b7280;
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  word-break: break-all;
}

.mb0 {
  margin-bottom: 0;
}

@media (max-width: 920px) {
  .user-summary,
  .yybgo-account {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 560px) {
  .user-topbar {
    align-items: flex-start;
    flex-direction: column;
    gap: 10px;
    padding: 12px 16px;
  }

  .user-content {
    width: min(100% - 24px, 1180px);
    padding-top: 12px;
  }
}
</style>
