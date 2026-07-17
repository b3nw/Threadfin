// Mirrors src/struct-webserver.go and src/struct-system.go on the Go side.

export interface ClientInfo {
  arch: string
  DVR: string
  epgSource: string
  errors: number
  'm3u-url': string
  os: string
  streams: string
  activeClients: number
  totalClients: number
  activePlaylist: number
  totalPlaylist: number
  uuid: string
  version: string
  warnings: number
  xepg: number
  'xepg-url': string
}

export interface WebScreenLog {
  errors: number
  log: string[]
  warnings: number
}

export interface PlaylistFile {
  name: string
  description?: string
  'file.source': string
  'file.threadfin'?: string
  'id.provider'?: string
  'last.update'?: string
  'provider.availability'?: number
  counter?: number
  tuner?: number
  buffer?: string
  type?: string
  'http_proxy.ip'?: string
  'http_proxy.port'?: string
  'http_headers.origin'?: string
  'http_headers.referer'?: string
  compatibility?: {
    streams?: number
    'group.title'?: number
    'tvg.id'?: number
    'stream.id'?: number
    'xmltv.channels'?: number
    'xmltv.programs'?: number
  }
  [key: string]: unknown
}

export interface Filter {
  active?: boolean
  liveEvent?: boolean
  caseSensitive?: boolean
  description?: string
  exclude?: string
  filter?: string
  include?: string
  name?: string
  type?: 'group-title' | 'custom-filter' | string
  startingNumber?: string
  'x-category'?: string
  [key: string]: unknown
}

export interface EpgChannel {
  'x-active': boolean
  'x-channelID': string
  'x-name': string
  'x-description'?: string
  'x-epg'?: string
  'x-group-title'?: string
  'x-category'?: string
  'x-xmltv-file': string
  'x-mapping': string
  'x-backup-channel-1'?: string
  'x-backup-channel-2'?: string
  'x-backup-channel-3'?: string
  'x-ppv-extra'?: string
  'x-update-channel-name'?: boolean
  'x-update-channel-icon'?: boolean
  'tvg-id'?: string
  'tvg-logo'?: string
  'group-title'?: string
  name?: string
  url?: string
  '_file.m3u.id'?: string
  '_file.m3u.name'?: string
  '_uuid.key'?: string
  [key: string]: unknown
}

export interface XmltvMapChannel {
  'display-name'?: string
  icon?: string
  [key: string]: unknown
}

export interface Settings {
  api: boolean
  'authentication.api': boolean
  'authentication.m3u': boolean
  'authentication.pms': boolean
  'authentication.web': boolean
  'authentication.xml': boolean
  'backup.keep': number
  'backup.path': string
  buffer: string
  'buffer.size.kb': number
  'buffer.timeout': number
  'cache.images': boolean
  epgSource: string
  'ffmpeg.options': string
  'ffmpeg.path': string
  'ffmpeg.forceHttp': boolean
  'vlc.options': string
  'vlc.path': string
  files: {
    hdhr: Record<string, PlaylistFile>
    m3u: Record<string, PlaylistFile>
    xmltv: Record<string, PlaylistFile>
  }
  'files.update': boolean
  filter: Record<string, Filter>
  language: string
  'log.entries.ram': number
  'mapping.first.channel': number
  port: string
  ssdp: boolean
  'temp.path': string
  tuner: number
  update: string[]
  'user.agent': string
  uuid: string
  udpxy: string
  version: string
  'xepg.replace.missing.images': boolean
  'xepg.replace.channel.title': boolean
  ThreadfinAutoUpdate: boolean
  storeBufferInRAM: boolean
  forceHttps: boolean
  excludeStreamHttps: boolean
  httpsPort: number
  bindIpAddress: string
  httpsThreadfinDomain: string
  httpThreadfinDomain: string
  enableNonAscii: boolean
  epgCategories: string
  epgCategoriesColors: string
  dummy: boolean
  dummyChannel: string
  ignoreFilters: boolean
  [key: string]: unknown
}

export interface UserRecord {
  data: {
    username?: string
    'authentication.web'?: boolean
    'authentication.pms'?: boolean
    'authentication.m3u'?: boolean
    'authentication.xml'?: boolean
    'authentication.api'?: boolean
    [key: string]: unknown
  }
  [key: string]: unknown
}

export interface ProbeInfo {
  resolution?: string
  frameRate?: string
  audioChannel?: string
}

export interface ServerResponse {
  clientInfo?: ClientInfo
  data?: {
    playlist?: { m3u?: { groups?: { text: string[]; value: string[] } } }
    StreamPreviewUI?: { activeStreams: string[]; inactiveStreams: string[] }
  }
  alert?: string
  configurationWizard?: boolean
  err?: string
  log?: WebScreenLog
  logoURL?: string
  openLink?: string
  openMenu?: string
  reload?: boolean
  settings?: Settings
  status: boolean
  token?: string
  users?: Record<string, UserRecord>
  wizard?: number
  xepg?: {
    epgMapping?: Record<string, EpgChannel>
    xmltvMap?: Record<string, Record<string, XmltvMapChannel>>
  }
  probeInfo?: ProbeInfo
  notification?: Record<
    string,
    { headline: string; message: string; new: boolean; time: string; type: string }
  >
}

export type Command =
  | 'getServerConfig'
  | 'updateLog'
  | 'saveSettings'
  | 'saveFilesM3U'
  | 'updateFileM3U'
  | 'saveFilesHDHR'
  | 'updateFileHDHR'
  | 'saveFilesXMLTV'
  | 'updateFileXMLTV'
  | 'saveFilter'
  | 'saveEpgMapping'
  | 'saveUserData'
  | 'saveNewUser'
  | 'resetLogs'
  | 'ThreadfinBackup'
  | 'ThreadfinRestore'
  | 'uploadLogo'
  | 'saveWizard'
  | 'probeChannel'
