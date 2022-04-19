package main

import "C"
import (
	"encoding/binary"
	"fmt"
	"gohookwechat/msg"
	"google.golang.org/protobuf/proto"
	"net"
	"os"
	"syscall"
	"unsafe"
)

/*
#cgo windows CFLAGS: -D X86=1
#include <stdio.h>
#include <wchar.h>

void Printf(const wchar_t *s) {
    printf("%S\n", s);
}
*/
import "C"
import (
	"github.com/jchv/go-webview2"
	"github.com/yamnikov-oleg/w32"
	"log"
	"path"
)

func InjectDll(handle uint32) {
	//maindir, _ := os.Getwd()
	//tmpdllpath := filepath.Join(maindir, "depency", "main.dll")

	tmpdllpath := "C:\\Users\\fengchuan\\source\\repos\\libwechat\\Release\\libwechat.dll"
	//tmpdllpath = "C:\\Users\\fengchuan\\GolandProjects\\wechathook\\main.dll"
	pathsize := C.wcslen((*C.ushort)(unsafe.Pointer(syscall.StringToUTF16Ptr(tmpdllpath))))
	dllpath := unsafe.Pointer(syscall.StringToUTF16Ptr(tmpdllpath))
	tmphandle := w32.OpenProcess(0x000F0000|0x00100000|0xFFFF, false, handle)
	if tmphandle == 0 {
		fmt.Println("open process failed")
	}
	addrress := w32.VirtualAllocEx(uint32(tmphandle), uint32(pathsize*2))
	w32.WriteProcessMemory(w32.HANDLE(tmphandle), addrress, dllpath, uint32(pathsize*2))
	h, _ := syscall.LoadLibrary("kernel32.dll")
	loadaddr, _ := syscall.GetProcAddress(h, "LoadLibraryW")
	w32.CreateRemoteThread(tmphandle, uintptr(loadaddr), uintptr(addrress))
	w32.CloseHandle(tmphandle)

}

var con net.Conn

func Server() {
	netListen, _ := net.Listen("tcp", "localhost:6666")
	defer netListen.Close()
	for {
		con, _ = netListen.Accept()
	}
}
func Sendmsg(wxid string, content string) {

	send := &msg.Sendmsg{Wxid: wxid, Content: content}
	pack := msg.Msg{Msgid: msg.Msg_SENDMSG, Payload: &msg.Msg_Sendmsg{Sendmsg: send}}
	fmt.Println(pack)
	pdata, _ := proto.Marshal(&pack)
	//var l uint32
	//l := uint32(len(pdata))
	//fmt.Println("datalen:", l)
	a := make([]byte, 4)
	binary.LittleEndian.PutUint32(a, uint32(len(pdata)))
	con.Write(a)
	con.Write(pdata)
}
func main() {
	go Server()
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     true,
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title: "go hook wechat",
		},
	})
	if w == nil {
		log.Fatalln("Failed to load webview.")
	}
	defer w.Destroy()
	w.SetSize(800, 600, webview2.HintFixed)
	w.Navigate("http://192.168.0.245:8080/")
	w.Bind("lunchwechat", func() {
		key, ok := w32.RegOpenKeyEx(w32.HKEY_CURRENT_USER, "Software\\Tencent", w32.KEY_QUERY_VALUE)
		if !ok {
			w32.MessageBox(0, "打开注册表失败", "打开注册表失败", 0)
			os.Exit(0)
		}
		defer w32.RegCloseKey(key)
		wechatdir := w32.RegGetString(key, "WeChat", "InstallPath")
		fmt.Println(wechatdir)
		startupinfo := &w32.STARTUPINFO{}
		processinfo := &w32.PROCESS_INFORMATION{}
		flag := w32.CreateProcess(path.Join(wechatdir, "WeChat.exe"), w32.CREATE_SUSPENDED, startupinfo, processinfo)
		if !flag {
			fmt.Println("打开微信失败")
		}
		w32.ResumeThread(processinfo.Thread)
		syscall.WaitForSingleObject(syscall.Handle(processinfo.Process), 1000)
		InjectDll(processinfo.ProcessId)
	})
	w.Bind("sendmsg", Sendmsg)
	w.Run()
}
