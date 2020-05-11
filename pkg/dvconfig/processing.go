// Package dvconfig manages configuration for http server
// MicroCore Copyright 2020 - 2020 by Danyil Dobryvechir (dobrivecher@yahoo.com ddobryvechir@gmail.com)
package dvconfig

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/Dobryvechir/microcore/pkg/dvcom"
	"github.com/Dobryvechir/microcore/pkg/dvmeta"
	"github.com/Dobryvechir/microcore/pkg/dvmodules"
	"github.com/Dobryvechir/microcore/pkg/dvparser"
	"github.com/Dobryvechir/microcore/pkg/dvprocessors"
	"github.com/Dobryvechir/microcore/pkg/dvproviders"
)

func prepareComRewriteMap(rewrites []DvRewrite) dvmeta.RewriteMap {
	var rewriteResult = make(dvmeta.RewriteMap)
	for _, c := range rewrites {
		url := c.From
		if url != "" && url[0] == '/' {
			url = url[1:]
		}
		urls := strings.Split(url, "/")
		full := true
		k := urls[0]
		url = "/" + url
		if strings.HasSuffix(url, "*") {
			full = false
			url = url[:len(url)-1]
		}
		rewriteResult[k] = append(rewriteResult[k], dvcom.GetRewriteMapItem(url, full, c.To))
	}
	return rewriteResult
}

func prepareMapOfStringArrays(smap map[string]string) map[string][]string {
	r := make(map[string][]string)
	for k, v := range smap {
		r[k] = []string{v}
	}
	return r
}

func prepareMicroCoreInfo(server *DvHostServer) *dvmeta.MicroCoreInfo {
	server.AccessControlMaxAge = strings.TrimSpace(server.AccessControlMaxAge)
	if server.AccessControlMaxAge != "" {
		if _, err := strconv.Atoi(server.AccessControlMaxAge); err != nil {
			log.Printf("accessControlMaxAge is expected to be either empty or number but it is %s \n", server.AccessControlMaxAge)
			server.AccessControlMaxAge = ""
		}
	}
	server.AccessControlAllowCredentials = strings.TrimSpace(server.AccessControlAllowCredentials)
	if server.AccessControlAllowCredentials != "" {
		if server.AccessControlAllowCredentials == "false" {
			server.AccessControlAllowCredentials = ""
		} else if server.AccessControlAllowCredentials != "true" {
			log.Printf("accessControlAllowCredentials is expected to be either empty or true or false but it is %s \n", server.AccessControlAllowCredentials)
			server.AccessControlAllowCredentials = ""
		}
	}
	dvServerInfo := &dvmeta.MicroCoreInfo{
		BaseFolderUrl:             dvcom.GetPurePath(server.BaseFolder),
		ExtraServerUrl:            dvcom.GetPurePath(server.ExtraServer),
		ExtraServerSettings:       server.ExtraServerSettings,
		ProxyName:                 dvcom.PrepareProxyName(server.ProxyName),
		BaseRewrite:               prepareComRewriteMap(server.Rewrites),
		ServerRewrite:             prepareComRewriteMap(server.ServerRewrites),
		HeadersStatic:             prepareMapOfStringArrays(server.HeadersStatic),
		HeadersStaticOptions:      prepareMapOfStringArrays(server.HeadersStaticOptions),
		HeadersExtraServer:        prepareMapOfStringArrays(server.HeadersExtraServer),
		HeadersExtraServerOptions: prepareMapOfStringArrays(server.HeadersExtraServerOptions),
		HeadersSpecial:            make(map[string]dvmeta.MicroCoreHeaderAttribute),
		HeadersSpecialOptions:     make(map[string]dvmeta.MicroCoreHeaderAttribute),
		HeadersSpecialStatic:      make(map[string]dvmeta.MicroCoreHeaderAttribute),
		DirectoryIndex:            dvparser.ConvertToNonEmptyList(server.DirectoryIndex),
		BaseProcessorBlocks:       dvprocessors.InitializeProcessors(server.Processors),
		BaseProviderBlocks:        dvproviders.MakeProviderBlocks(server.Providers),
		PostProcessorBlocks:       dvprocessors.InitializePostProcessors(server.PostProcessors),
		HostHeader:                strings.TrimSpace(server.HostHeader),
	}

	accessControlAllowOrigin := dvcom.PrepareAccessControlLists(server.AccessControlAllowOrigin)
	accessControlAllowMethods := dvcom.PrepareAccessControlLists(server.AccessControlAllowMethod)
	accessControlAllowHeaders := dvcom.PrepareAccessControlLists(server.AccessControlAllowHeaders)
	accessControlExposeHeaders := dvcom.PrepareAccessControlLists(server.AccessControlExposeHeaders)
	if server.AccessControlMaxAge != "" {
		dvServerInfo.HeadersStaticOptions["Access-Control-Max-Age"] = []string{server.AccessControlMaxAge}
		dvServerInfo.HeadersExtraServerOptions["Access-Control-Max-Age"] = []string{server.AccessControlMaxAge}
	}
	if server.AccessControlAllowCredentials != "" {
		dvServerInfo.HeadersStatic["Access-Control-Allow-Credentials"] = []string{"true"}
		dvServerInfo.HeadersStaticOptions["Access-Control-Allow-Credentials"] = []string{"true"}
		dvServerInfo.HeadersExtraServer["Access-Control-Allow-Credentials"] = []string{"true"}
		dvServerInfo.HeadersExtraServerOptions["Access-Control-Allow-Credentials"] = []string{"true"}
	}
	cacheControl := strings.TrimSpace(server.CacheControl)
	if cacheControl != "" {
		dvServerInfo.HeadersStatic["Cache-Control"] = []string{cacheControl}
		dvServerInfo.HeadersExtraServer["Cache-Control"] = []string{cacheControl}
	}
	if accessControlExposeHeaders.Kind >= 0 {
		dvServerInfo.HeadersStatic["Access-Control-Expose-Headers"] = []string{accessControlExposeHeaders.Plain}
		dvServerInfo.HeadersSpecial["Access-Control-Expose-Headers"] = accessControlExposeHeaders
	}
	if accessControlAllowHeaders.Kind >= 0 {
		dvServerInfo.HeadersStaticOptions["Access-Control-Allow-Headers"] = []string{accessControlAllowHeaders.Plain}
		dvServerInfo.HeadersSpecialOptions["Access-Control-Allow-Headers"] = accessControlAllowHeaders
	}
	if accessControlAllowMethods.Kind >= 0 {
		dvServerInfo.HeadersStaticOptions["Access-Control-Allow-Methods"] = []string{}
		dvServerInfo.HeadersSpecialOptions["Access-Control-Allow-Methods"] = accessControlAllowMethods
	}
	if accessControlAllowOrigin.Kind >= 0 {
		accessControlAllowOrigin.Kind = dvcom.HEADERS_SET_ORIGIN
		if _, ok := accessControlAllowOrigin.Imap["*"]; ok {
			accessControlAllowOrigin.Kind = dvcom.HEADERS_SET_ORIGIN_ALWAYS
		}
		dvServerInfo.HeadersSpecial["Access-Control-Allow-Origin"] = accessControlAllowOrigin
		dvServerInfo.HeadersSpecialOptions["Access-Control-Allow-Origin"] = accessControlAllowOrigin
		dvServerInfo.HeadersSpecialStatic["Access-Control-Allow-Origin"] = accessControlAllowOrigin
	}
	dvServerInfo.ModuleHandler = dvmodules.RegisterEndPointHandlers(server.Modules)
	dvServerInfo.ActionHandler = dvmodules.RegisterEndPointActions(server.Actions)
	return dvServerInfo
}

// ProcessBaseFolder sets up the http server using information for all requested servers
func ProcessBaseFolder(server *DvHostServer, hostServers []DvHostServer) {
	if server != nil || len(hostServers) > 0 {
		defaultServerInfo := prepareMicroCoreInfo(server)
		hostServerInfo := make(map[string]*dvmeta.MicroCoreInfo)
		for i, c := range hostServers {
			hosts := strings.Split(strings.TrimSpace(c.Hosts), " ")
			if len(hosts) == 0 || hosts[0] == "" {
				log.Printf("HostsServer[%d] is omitted because not filled", i)
				continue
			}
			serverInfo := prepareMicroCoreInfo(&c)
			for _, host := range hosts {
				if _, ok := hostServerInfo[host]; ok {
					log.Printf("HostServer host %s is duplicated", host)
					continue
				}
				hostServerInfo[host] = serverInfo
			}
		}
		http.HandleFunc("/", dvcom.MakeDefaultHandler(defaultServerInfo, hostServerInfo))
	}
}
