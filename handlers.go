package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	controller "github.com/gravitl/netmaker/controllers"
	"github.com/gravitl/netmaker/dnslogic"
	"github.com/gravitl/netmaker/functions"
	"github.com/gravitl/netmaker/models"
)

func ProcessLogin(c *gin.Context) {
	fmt.Println("Processing Login")
	var AuthRequest models.UserAuthParams
	AuthRequest.UserName = c.PostForm("user")
	AuthRequest.Password = c.PostForm("pass")
	session := sessions.Default(c)
	//don't need the jwt
	_, err := controller.VerifyAuthRequest(AuthRequest)
	if err != nil {
		fmt.Println("error verifying AuthRequest: ", err)
		session.Set("message", err.Error())
		session.Set("loggedIn", false)
		c.HTML(http.StatusUnauthorized, "Login", gin.H{"message": err})
	} else {
		session.Set("loggedIn", true)
		//init message
		session.Set("message", "")
		session.Options(sessions.Options{MaxAge: 28800})
		user, err := controller.GetUser(AuthRequest.UserName)
		if err != nil {
			fmt.Println("err retrieving user: ", err)
		}
		session.Set("username", user.UserName)
		session.Set("isAdmin", user.IsAdmin)
		session.Set("networks", user.Networks)
		session.Save()
		location := url.URL{Path: "/"}
		c.Redirect(http.StatusFound, location.RequestURI())
	}
}

func NewUser(c *gin.Context) {
	var user models.User
	user.UserName = c.PostForm("user")
	user.Password = c.PostForm("pass")
	user.IsAdmin = true
	hasAdmin, err := controller.HasAdmin()
	if err != nil {
		ReturnError(c, http.StatusInternalServerError, err, "")
		return
	}
	if hasAdmin {
		ReturnError(c, http.StatusUnauthorized, errors.New("Admin Exists"), "")
		return
	}
	_, err = controller.CreateUser(user)
	if err != nil {
		ReturnError(c, http.StatusUnauthorized, err, "")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func DisplayLanding(c *gin.Context) {
	var data PageData
	page := ""
	session := sessions.Default(c)
	if session.Get("page") != nil {
		page = session.Get("page").(string)
	}
	fmt.Println("Initialializing PageData for page ", page)
	if page != "" {
		data.Init(page, c)
	} else {
		data.Init("Networks", c)
	}
	c.HTML(http.StatusOK, "layout", data)
}

func CreateNetwork(c *gin.Context) {
	var net models.Network

	net.NetID = c.PostForm("name")
	net.AddressRange = c.PostForm("address")
	net.IsDualStack = c.PostForm("dual")
	net.IsLocal = c.PostForm("local")
	net.DefaultUDPHolePunch = c.PostForm("udp")

	err := controller.CreateNetwork(net)

	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func EditNetwork(c *gin.Context) {
	network := c.PostForm("network")
	fmt.Println("editing network ", network)
	var Data models.Network
	Data, err := controller.GetNetwork(network)
	if err != nil {
		fmt.Println("error getting net details \n", err)
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	c.HTML(http.StatusOK, "EditNet", Data)
}

func DeleteNetwork(c *gin.Context) {
	network := c.PostForm("network")
	err := controller.DeleteNetwork(network)
	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func UpdateNetwork(c *gin.Context) {
	var network models.Network
	net := c.PostForm("NetID")
	network.NetID = c.PostForm("NetID")
	network.AddressRange = c.PostForm("Address")
	network.LocalRange = c.PostForm("Local")
	network.DisplayName = c.PostForm("Name")
	network.DefaultInterface = c.PostForm("Interface")
	port, err := strconv.Atoi(c.PostForm("Port"))
	if err != nil {
		fmt.Println("error converting port", err)
	}
	network.DefaultListenPort = int32(port)
	network.DefaultPostUp = c.PostForm("PostUp")
	network.DefaultPostDown = c.PostForm("PostDown")
	keep, err := strconv.Atoi(c.PostForm("Keepalive"))
	if err != nil {
		fmt.Println("error converting keepalive", err)
	}
	network.DefaultKeepalive = int32(keep)
	check, err := strconv.Atoi(c.PostForm("CheckinInterval"))
	if err != nil {
		fmt.Println("error converting check interval", err)
	}
	network.DefaultCheckInInterval = int32(check)
	network.IsDualStack = c.PostForm("DualStack")
	if network.IsDualStack == "" {
		network.IsDualStack = "no"
	}
	network.DefaultSaveConfig = c.PostForm("DefaultSaveConfig")
	if network.DefaultSaveConfig == "" {
		network.DefaultSaveConfig = "no"
	}
	network.DefaultUDPHolePunch = c.PostForm("UDPHolePunching")
	if network.DefaultUDPHolePunch == "" {
		network.DefaultUDPHolePunch = "no"
	}
	network.AllowManualSignUp = c.PostForm("AllowManualSignup")
	if network.AllowManualSignUp == "" {
		network.AllowManualSignUp = "no"
	}

	if network.LocalRange == "" {
		network.IsLocal = "no"
	} else {
		network.IsLocal = "yes"
	}
	if network.AddressRange6 == "" {
		network.IsIPv6 = "no"
	} else {
		network.IsIPv6 = "yes"
	}
	if network.AddressRange == "" {
		network.IsIPv4 = "no"
	} else {
		network.IsIPv4 = "yes"
	}
	network.IsGRPCHub = "no"

	oldnetwork, err := controller.GetNetwork(net)
	if err != nil {
		fmt.Println("error getting network ", err)
	}
	updaterange, updatelocal, err := oldnetwork.Update(&network)
	if err != nil {
		fmt.Println("error updating network ", err)
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	if updaterange {
		err = functions.UpdateNetworkNodeAddresses(network.NetID)
		if err != nil {
			fmt.Println("error updating network Node Addresses", err)
			ReturnError(c, http.StatusBadRequest, err, "Networks")
			return
		}
	}
	if updatelocal {
		err = functions.UpdateNetworkLocalAddresses(network.NetID)
		if err != nil {
			fmt.Println("error updating network Local Addresses", err)
			ReturnError(c, http.StatusBadRequest, err, "Networks")
			return
		}
	}

	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func RefreshKeys(c *gin.Context) {
	net := c.Param("net")
	_, err := controller.KeyUpdate(net)
	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Keys")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func NewKey(c *gin.Context) {
	var key models.AccessKey
	var err error
	net := c.PostForm("network")
	key.Name = c.PostForm("name")
	key.Uses, err = strconv.Atoi(c.PostForm("uses"))
	if err != nil {
		key.Uses = 1
	}
	network, err := controller.GetNetwork(net)
	if err != nil {
		fmt.Println("error retrieving network ", err)
		ReturnError(c, http.StatusBadRequest, err, "Keys")
		return
	}
	_, err = controller.CreateAccessKey(key, network)
	if err != nil {
		fmt.Println("error creating key", err)
		ReturnError(c, http.StatusBadRequest, err, "Keys")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func DeleteKey(c *gin.Context) {
	net := c.PostForm("net")
	name := c.PostForm("key")
	fmt.Println("Delete Key params: ", net, name)
	if err := controller.DeleteKey(name, net); err != nil {
		fmt.Println("error deleting key", err)
		ReturnError(c, http.StatusBadRequest, err, "Keys")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func LogOut(c *gin.Context) {
	session := sessions.Default(c)
	session.Set("loggedIn", false)
	session.Save()
	fmt.Println("User Logged Out", session.Get("loggedIn"))
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func CreateUser(c *gin.Context) {
	var user models.User
	fmt.Println("creating new user")
	user.UserName = c.PostForm("user")
	user.Password = c.PostForm("pass")
	if c.PostForm("admin") == "true" {
		user.IsAdmin = true
	} else {
		user.IsAdmin = false
	}
	user.Networks, _ = c.GetPostFormArray("network[]")
	fmt.Println("networks: ", user.Networks)
	_, err := controller.CreateUser(user)
	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}

	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func DeleteUser(c *gin.Context) {
	user := c.PostForm("user")
	success, err := controller.DeleteUser(user)
	if !success {
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func EditUser(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username").(string)
	user, err := controller.GetUser(username)
	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	c.HTML(http.StatusOK, "EditUser", user)
}

func UpdateUser(c *gin.Context) {
	var new models.User
	username := c.Param("user")
	user, err := controller.GetUserInternal(username)
	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	new.UserName = c.PostForm("username")
	new.Password = c.PostForm("password")
	_, err = controller.UpdateUser(new, user)
	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Networks")
		return
	}
	session := sessions.Default(c)
	session.Set("message", "user has been updated")
	session.Save()
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func EditNode(c *gin.Context) {
	network := c.PostForm("network")
	mac := c.PostForm("mac")
	var node models.Node
	node, err := controller.GetNode(mac, network)
	if err != nil {
		fmt.Println("error getting node details \n", err)
		ReturnError(c, http.StatusBadRequest, err, "Nodes")
		return
	}
	c.HTML(http.StatusOK, "EditNode", node)
}

func DeleteNode(c *gin.Context) {
	mac := c.PostForm("mac")
	net := c.PostForm("net")
	fmt.Println("deleting node ", mac, net)
	err := controller.DeleteNode(mac+"###"+net, false)
	if err != nil {
		ReturnError(c, http.StatusBadRequest, err, "Nodes")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func UpdateNode(c *gin.Context) {

	var node *models.Node
	if err := c.ShouldBind(&node); err != nil {
		fmt.Println("should bind")
		ReturnError(c, http.StatusBadRequest, err, "Nodes")
		return
	}
	net := c.Param("net")
	mac := c.Param("mac")
	fmt.Printf("=============%T %T %T %v %v %v", net, mac, node, net, mac, node)
	oldnode, err := functions.GetNodeByMacAddress(net, mac)
	if err != nil {
		fmt.Println("Get node with mac ", mac, " and Network ", net)
		ReturnError(c, http.StatusBadRequest, err, "Nodes")
		return
	}
	if err = oldnode.Update(node); err != nil {
		fmt.Println("update network")
		ReturnError(c, http.StatusBadRequest, err, "Nodes")
		return
	}
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())

}

func NodeHealth(c *gin.Context) {
	nodes, err := models.GetAllNodes()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var response []NodeStatus
	var nodeHealth NodeStatus
	for _, node := range nodes {
		nodeHealth.Mac = node.MacAddress
		nodeHealth.Network = node.Network
		lastupdate := time.Now().Sub(time.Unix(node.LastCheckIn, 0))
		if lastupdate.Minutes() > 15.0 {
			nodeHealth.Status = "Error: Node last checked in more than 15 minutes ago: "
			nodeHealth.Color = "w3-deep-orange"
		} else if lastupdate.Minutes() > 5.0 {
			nodeHealth.Status = "Warning: Node last checked in more than 5 minutes ago: "
			nodeHealth.Color = "w3-khaki"
		} else {
			nodeHealth.Status = "Healthy: Node checked in within the last 5 minutes: "
			nodeHealth.Color = "w3-teal"
		}
		response = append(response, nodeHealth)
	}
	c.JSON(http.StatusOK, response)
	return
}

func CreateDNS(c *gin.Context) {
	var entry models.DNSEntry
	entry.Network = c.PostForm("network")
	entry.Name = c.PostForm("name")
	entry.Address = c.PostForm("address")
	if err := controller.ValidateDNSCreate(entry); err != nil {
		fmt.Println("validation err dns: ", err)
		ReturnError(c, http.StatusBadRequest, err, "DNS")
		return
	}
	_, err := controller.CreateDNS(entry)
	if err != nil {
		fmt.Println("err dns: ", err)
		ReturnError(c, http.StatusBadRequest, err, "DNS")
		return
	}
	session := sessions.Default(c)
	session.Set("message", "DNS Entry Created")
	session.Set("page", "DNS")
	session.Save()
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func DeleteDNS(c *gin.Context) {
	network := c.Param("net")
	name := c.Param("name")
	if err := controller.DeleteDNS(name, network); err != nil {
		fmt.Println("err dns delete", err)
		ReturnError(c, http.StatusBadRequest, err, "DNS")
		return
	}
	if err := dnslogic.SetDNS(); err != nil {
		fmt.Println("err set dns", err)
		ReturnError(c, http.StatusBadRequest, err, "DNS")
		return
	}
	session := sessions.Default(c)
	session.Set("message", "DNS Entry Deleted")
	session.Set("page", "DNS")
	session.Save()
	location := url.URL{Path: "/"}
	c.Redirect(http.StatusFound, location.RequestURI())
}

func ReturnError(c *gin.Context, status int, err error, page string) {
	var data PageData
	session := sessions.Default(c)
	session.Set("message", err.Error())
	session.Set("page", page)
	session.Save()
	if page != "" {
		data.Init(page, c)
	} else {
		data.Init("Networks", c)
	}
	c.HTML(status, "layout", data)
}
