package startup

import (
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/providers/confmap"
	cp "github.com/otiai10/copy"
	log "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

func mountDir(host, guest string) error {
	log.Debugf("mounting %s to %s\n", host, guest)
	return syscall.Mount("none", guest, "hostfs", 0, host)
}

type VConf struct {
	Quiet        bool              `koanf:"quiet"`
	Driver       string            `koanf:"driver"`
	HostName     string            `koanf:"name"`
	HostLab      bool              `koanf:"hostlab"`
	DefaultRoute string            `koanf:"def_route"`
	Interfaces   map[string]string `koanf:"interfaces"`
	Volumes      map[string]string `koanf:"volumes"`
}

func parseCmdline() (conf VConf, err error) {
	clBytes, err := os.ReadFile("/proc/cmdline")
	if err != nil {
		return conf, err
	}
	cl := strings.Fields(string(clBytes))
	opts := make(map[string]interface{}, 0)
	for _, o := range cl {
		if o == "quiet" {
			opts["quiet"] = true
			continue
		}
		if strings.HasPrefix(o, "name=") && len(o) > 5 {
			opts["name"] = o[5:]
		}
		if !strings.HasPrefix(o, "kstart:") {
			continue
		}
		o := o[7:]
		kv := strings.Split(o, "=")
		if len(kv) != 2 {
			return conf, fmt.Errorf("option %s has incorrect format", o)
		}
		opts[kv[0]] = kv[1]
	}
	k := koanf.New(".")
	err = k.Load(confmap.Provider(opts, "."), nil)
	if err != nil {
		return conf, err
	}
	err = k.Unmarshal("", &conf)
	log.Debugf("parsed cmdline: %+v\n", conf)
	return conf, err
}

func loadEnv() (conf VConf, err error) {
	envBytes, err := os.ReadFile("/proc/1/environ")
	if err != nil {
		return conf, err
	}
	envVars := strings.Split(string(envBytes), "\000")
	opts := make(map[string]interface{}, 0)
	for _, v := range envVars {
		if !strings.HasPrefix(v, "kstart-") || len(v) < 8 {
			continue
		}
		kv := strings.Split(v[7:], "=")
		if len(kv) != 2 {
			return conf, fmt.Errorf("option %s has incorrect format", v)
		}
		opts[kv[0]] = kv[1]
	}
	conf.HostName, err = os.Hostname()
	if err != nil {
		return conf, err
	}
	k := koanf.New(".")
	err = k.Load(confmap.Provider(opts, "."), nil)
	if err != nil {
		return conf, err
	}

	err = k.Unmarshal("", &conf)
	log.Debugf("parsed env: %+v\n", conf)
	if conf.Driver != "UML" && conf.Driver != "podman" {
		return conf, fmt.Errorf("driver %s is not supported", conf.Driver)
	}
	return conf, err
}

func LoadConf() (conf VConf, err error) {
	cmdline, err := parseCmdline()
	if err != nil {
		return conf, err
	}
	if cmdline.Driver == "UML" {
		return cmdline, err
	}
	// not uml so driver is podman
	return loadEnv()
}

func hostname(name string) error {
	log.Debugf("setting hostname to %s\n", name)
	err := os.WriteFile("/etc/hostname", []byte(name), 0644)
	if err != nil {
		return err
	}
	return syscall.Sethostname([]byte(name))
}

func hostsFile(name string) error {
	origHostsBytes, err := os.ReadFile("/etc/hosts")
	if err != nil {
		return err
	}
	hostsContents := fmt.Sprintf("127.0.0.1 %s\n", name) + string(origHostsBytes)
	return os.WriteFile("/etc/hosts", []byte(hostsContents), 0644)
}

func setupHosts(name string) error {
	_, err := os.Stat("/etc/vhostconfigured")
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		log.Debug("hosts/hostname already configured")
		return nil
	}
	// error is file not exist, need to set up vhost
	if err := hostname(name); err != nil {
		return err
	}
	if err := hostsFile(name); err != nil {
		return err
	}
	return os.WriteFile("/etc/vhostconfigured", []byte(""), 0644)
}

func copyInFiles(hostname string) error {
	opts := cp.Options{
		PreserveOwner: false,
	}
	// copy in files from shared if exists
	_, err := os.Stat("/hostlab/shared")
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		log.Debug("copying in files from shared dir")
		err := cp.Copy("/hostlab/shared", "/", opts)
		if err != nil {
			return err
		}
	}
	// copy in files from machine dir if exists
	machineDir := fmt.Sprintf("/hostlab/%s", hostname)
	_, err = os.Stat(machineDir)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err == nil {
		log.Debug("copying in files from machine dir")
		err := cp.Copy(machineDir, "/", opts)
		if err != nil {
			return err
		}
	}
	return nil
}

func umlSetup(conf VConf) (err error) {
	if conf.HostLab {
		os.Mkdir("/hostlab", 0744)
		err = mountDir("/hostlab", "/hostlab")
		if err != nil {
			return fmt.Errorf("could not mount /hostlab: %w", err)
		}
	}
	os.Mkdir("/run/uml-guest", 0744)
	err = mountDir("/run/uml", "/run/uml-guest")
	if err != nil {
		return fmt.Errorf("could not mount /run/uml to /run/uml-guest: %w", err)
	}
	return nil
}

func setupIfaces(ifaces map[string]string, defRoute string) error {
	log.Debug("setting up interfaces: ", ifaces)
	for iface, addr := range ifaces {
		if addr == "" {
			continue
		}
		i, err := netlink.LinkByName(iface)
		if err != nil {
			log.Error(err)
			continue
		}
		a, err := netlink.ParseAddr(addr)
		if err != nil {
			log.Error(err)
			continue
		}
		err = netlink.AddrAdd(i, a)
		if err != nil {
			log.Error(err)
			continue
		}
	}
	if defRoute != "" {
		ia := strings.Split(defRoute, ":")
		if len(ia) != 2 {
			return fmt.Errorf("invalid def_route format %s\n", defRoute)
		}
		gwIp := net.ParseIP(ia[1])
		if gwIp == nil {
			return fmt.Errorf("def_route ip %s is not valid\n", defRoute)
		}
		link, err := netlink.LinkByName(ia[0])
		if err != nil {
			return fmt.Errorf("could not find interface %s for default route", ia[0])
		}
		err = netlink.LinkSetUp(link)
		if err != nil {
			return err
		}
		_, dst, _ := net.ParseCIDR("0.0.0.0/0")
		r := netlink.Route{
			LinkIndex: link.Attrs().Index,
			Scope:     netlink.SCOPE_UNIVERSE,
			Dst:       dst,
			Gw:        gwIp,
		}
		err = netlink.RouteAdd(&r)
		if err != nil {
			return fmt.Errorf("cannot set default route to %s: %w", defRoute, err)
		}
	}
	return nil
}

func renameVecs() error {
	ifaces, err := netlink.LinkList()
	if err != nil {
		return err
	}
	for _, i := range ifaces {
		name := i.Attrs().Name
		if strings.HasPrefix(name, "vec") && len(name) > 3 {
			suffix := name[3:]
			err = netlink.LinkSetName(i, "eth"+suffix)
			if err != nil {
				log.Error(err)
			}
			err = netlink.LinkSetAlias(i, name) // save original name as alias
			if err != nil {
				log.Error(err)
			}
		}
	}
	return nil
}

func StartPhaseOne() error {
	conf, err := LoadConf()
	if err != nil {
		return err
	}
	if conf.Driver == "UML" {
		log.Debug("setting up uml vm")
		err = umlSetup(conf)
		if err != nil {
			return err
		}
	} else {
		log.Debug("vm is not of type uml")
	}
	// TODO mount other volumes
	if conf.Driver == "UML" {
		err = renameVecs()
		if err != nil {
			return err
		}
	}
	err = setupIfaces(conf.Interfaces, conf.DefaultRoute)
	if err != nil {
		return err
	}
	err = setupHosts(conf.HostName)
	if err != nil {
		return err
	}
	err = copyInFiles(conf.HostName)
	if err != nil {
		return err
	}
	// TODO setup mgmt iface
	return nil
}

func Shutdown() error {
	err := syscall.Unmount("/hostlab", 0)
	if err != nil {
		log.Errorf("could not unmount /hostlab: %v", err)
	}
	err = syscall.Unmount("/run/uml-guest", 0)
	if err != nil {
		log.Errorf("could not unmount /run/uml-guest: %v", err)
	}
	conf, err := LoadConf()
	if err != nil {
		return err
	}
	for _, v := range conf.Volumes {
		err = syscall.Unmount(v, 0)
		if err != nil {
			log.Errorf("could not unmount %s: %v", v, err)
		}
	}
	// TODO run shutdown script (machine.shutdown and shared.shutdown)
	return nil
}

func SetLogLevel() error {
	// check proc cmdline
	return nil
}
