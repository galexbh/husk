package k8sclient

import "time"

// defaultDetectTimeout acota las llamadas de detección/discovery que no
// deben depender del contexto del caller (por ejemplo, detectOpenShift, que
// corre durante la construcción del Client).
const defaultDetectTimeout = 5 * time.Second
