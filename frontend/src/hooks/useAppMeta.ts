import { useEffect, useState } from 'react'
import {
  DEFAULT_ACCENT,
  DEFAULT_APP_NAME,
  getAppMeta,
  type AppMeta,
} from '../lib/pocketbase'

export function useAppMeta(): AppMeta {
  const [meta, setMeta] = useState<AppMeta>({
    appName: DEFAULT_APP_NAME,
    accent: DEFAULT_ACCENT,
  })

  useEffect(() => {
    let active = true

    void getAppMeta().then((next) => {
      if (active) {
        setMeta(next)
      }
    })

    return () => {
      active = false
    }
  }, [])

  return meta
}
