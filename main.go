/**
* This is an example plugin where we use both arguments and flags. The plugin
* will echo all arguments passed to it. The flag -uppercase will upcase the
* arguments passed to the command.
**/
package main

import (
	"fmt"
   "os"
   "io"
   "time"
   "os/exec"
   "bytes"
   "strings"

	"code.cloudfoundry.org/cli/plugin"
)

type PluginDemonstratingParams struct {
	uppercase *bool
}

func main() {
	plugin.Start(new(PluginDemonstratingParams))
}

func getOrgAndSpaceNames() (string, string, error) {
   cmd := exec.Command("cf", "target")
   buffer := bytes.NewBuffer(nil)
   cmd.Stdout = buffer
   if err := cmd.Run(); err != nil {
      return "", "", err
   }
   splitLines := strings.Split(string(buffer.Bytes()), "\n")
   var orgName string
   var spaceName string
   for _, line := range splitLines {
      if strings.Contains(line, "org") {
         orgName = strings.TrimSpace(strings.Split(line, ":")[1])
      }
      if strings.Contains(line, "space") {
         spaceName = strings.TrimSpace(strings.Split(line, ":")[1])
      }
   }
   return orgName, spaceName, nil
}

func checkIfAppExists(appName string) error {
   cmd := exec.Command("cf", "app", appName)
   if err := cmd.Run(); err != nil {
      return err
   }
   return nil
}

func (pluginDemo *PluginDemonstratingParams) Run(cliConnection plugin.CliConnection, args []string) {
   if len(args) != 3 {
      fmt.Println("Invalid parameters")
		os.Exit(1)
   }

   appName := args[1]
   instanceName := args[2]

   if err := checkIfAppExists(appName); err != nil {
      fmt.Println("App does not exist")
      os.Exit(1)
   }

   // Create a new binding
   cmd := exec.Command(
      "pdc",
      "binding",
      "create",
      "--name",
      fmt.Sprintf("%s-binding", appName),
      "--instance",
      instanceName,
   )
   eBuffer := bytes.NewBuffer(nil)
   cmd.Stderr = eBuffer
   err := cmd.Run()
   if err != nil {
      fmt.Printf("Binding failed: %s\n", string(eBuffer.Bytes()))
      os.Exit(1)
   }
   fmt.Println("Binding in progress...")

   // Wait for the binding to complete (fingers crossed)
   time.Sleep(8 * time.Second)

   // Convert the binding to the correct format
   c1 := exec.Command(
      "pdc",
      "binding",
      "get",
      "--name",
      fmt.Sprintf("%s-binding", appName),
   )
   c2 := exec.Command(
      "yq",
      "-rc",
      ".credentials",
   )
   r, w := io.Pipe()
   c1.Stdout = w
   c2.Stdin = r

   var b2 bytes.Buffer
   c2.Stdout = &b2

   c1.Start()
   c2.Start()
   c1.Wait()
   w.Close()
   c2.Wait()

   // Create a user provided service with the binding
   cmd = exec.Command(
      "cf",
      "cups",
      "managed-pdc-service",
      "-p",
      fmt.Sprintf("'%s'", string(b2.Bytes())),
   )
   eBuffer = bytes.NewBuffer(nil)
   cmd.Stderr = eBuffer
   err = cmd.Run()
   if err != nil {
      fmt.Printf("Creating UPS failed: %s\n", string(eBuffer.Bytes()))
      os.Exit(1)
   }

   // Bind to the app
   cmd = exec.Command(
      "cf",
      "bs",
      appName,
      "managed-pdc-service",
   )
   eBuffer = bytes.NewBuffer(nil)
   cmd.Stderr = eBuffer
   err = cmd.Run()
   if err != nil {
      fmt.Printf("Binding service failed: %s\n", string(eBuffer.Bytes()))
      os.Exit(1)
   }

}

func (pluginDemo *PluginDemonstratingParams) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "pdc-plugin",
		Version: plugin.VersionType{
			Major: 0,
			Minor: 1,
			Build: 1,
		},
		Commands: []plugin.Command{
			{
				Name:     "bind-pdc-service",
				HelpText: "Binds a service instance from the Pivotal Developer Console to an application running in Cloud Foundry",
				UsageDetails: plugin.Usage{
					Usage: "cf bind-pdc-service APP_NAME SERVICE_INSTANCE_NAME",
				},
			},
		},
	}
}
