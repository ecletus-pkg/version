package version

import (
	"github.com/ecletus/plug"
	"github.com/moisespsena/go-path-helpers"
)

var (
	pkg        = path_helpers.GetCalledDir()
	E_REGISTER = pkg + ".register"
)

type VersionRegisterEvent struct {
	plug.PluginEventInterface
	versions *map[string]*Version
}

func (e *VersionRegisterEvent) Set(name string, version *Version) {
	(*e.versions)[name] = version
}

func OnRegister(dis plug.EventDispatcherInterface, cb func(e *VersionRegisterEvent)) {
	dis.On(E_REGISTER, func(e plug.EventInterface) {
		cb(e.(*VersionRegisterEvent))
	})
}

func triggerRegister(dis plug.PluginEventDispatcherInterface, versions *map[string]*Version) {
	_ = dis.TriggerPlugins(&VersionRegisterEvent{plug.NewPluginEvent(E_REGISTER), versions})
}
