package main

import (
    "runtime"
    "github.com/go-gl/mathgl/mgl32"
    "fmt"
    "github.com/go-gl/gl/v3.3-core/gl"
    "github.com/go-gl/glfw/v3.3/glfw"
)

// Constants and global variables

const (
    g_WindowWidth  = 1000
    g_WindowHeight = 1000
)

const g_WindowTitle  = "Basic Go OpenGL example application"
var g_ShaderID uint32
var g_ComputeShaderID uint32

// Normal Camera
var g_fovy      = mgl32.DegToRad(90.0)
var g_aspect    = float32(g_WindowWidth)/g_WindowHeight
var g_nearPlane = float32(1.0)
var g_farPlane  = float32(200.0)

// Light camera
var g_smWidth, g_smHeight int32 = 480, 480
var g_lightFovy      = mgl32.DegToRad(90.0)
var g_lightAspect    = float32(g_smWidth)/float32(g_smHeight)
var g_lightNearPlane = float32(1.0)
var g_lightFarPlane  = float32(200.0)
var g_lightViewProjectionMat mgl32.Mat4
var g_varianceDepthShader uint32
var g_lightFbo, g_lightColorTex, g_lightDepthTex uint32

var g_viewMatrix          mgl32.Mat4

var g_shadowEnabled bool = true
var g_boxBlurCompute uint32
var g_lightColorTexBlur1 uint32
var g_blurKernelSize int32 = 5

var g_objects []Object
var g_light Object

var g_multisamplingEnabled bool = true

var g_msFbo, g_msColorTex, g_msDepthTex uint32



var g_timeSum float32 = 0.0
var g_lastCallTime float64 = 0.0
var g_frameCount int = 0
var g_fps float32 = 60.0

var g_fillMode = 0

func init() {
    // GLFW event handling must run on the main OS thread
    runtime.LockOSThread()
}


func printHelp() {
    fmt.Println(
        `Help yourself.`,
    )
}

// Set OpenGL version, profile and compatibility
func initGraphicContext() (*glfw.Window, error) {
    glfw.WindowHint(glfw.Resizable, glfw.True)
    glfw.WindowHint(glfw.ContextVersionMajor, 4)
    glfw.WindowHint(glfw.ContextVersionMinor, 3)
    glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
    glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

    window, err := glfw.CreateWindow(g_WindowWidth, g_WindowHeight, g_WindowTitle, nil, nil)
    if err != nil {
        return nil, err
    }
    window.MakeContextCurrent()

    // Initialize Glow
    if err := gl.Init(); err != nil {
        return nil, err
    }

    return window, nil
}

func defineModelMatrix(shader uint32, pos, scale mgl32.Vec3) {
    matScale := mgl32.Scale3D(scale.X(), scale.Y(), scale.Z())
    matTrans := mgl32.Translate3D(pos.X(), pos.Y(), pos.Z())
    model := matTrans.Mul4(matScale)
    modelUniform := gl.GetUniformLocation(shader, gl.Str("modelMat\x00"))
    gl.UniformMatrix4fv(modelUniform, 1, false, &model[0])
}

// Defines the Model-View-Projection matrices for the shader.
func defineMatrices(shader uint32) {
    projection := mgl32.Perspective(g_fovy, g_aspect, g_nearPlane, g_farPlane)
    camera := mgl32.LookAtV(GetCameraLookAt())

    viewProjection := projection.Mul4(camera);
    cameraUniform := gl.GetUniformLocation(shader, gl.Str("viewProjectionMat\x00"))
    gl.UniformMatrix4fv(cameraUniform, 1, false, &viewProjection[0])
}

// Defines the Model-View-Projection matrices for the shader.
func defineLightMatrices(shader uint32) {
    projection := mgl32.Perspective(g_lightFovy, g_lightAspect, g_lightNearPlane, g_lightFarPlane)
    //projection := mgl32.Ortho(-50, 50, -50, 50, g_nearPlane, g_farPlane)
    camera := mgl32.LookAtV(g_light.Pos, mgl32.Vec3{0,0,0}, mgl32.Vec3{0,1,0})

    g_lightViewProjectionMat = projection.Mul4(camera)
    cameraUniform := gl.GetUniformLocation(shader, gl.Str("viewProjectionMat\x00"))
    gl.UniformMatrix4fv(cameraUniform, 1, false, &g_lightViewProjectionMat[0])

}

func renderObject(shader uint32, obj Object) {

    // Model transformations are now encoded per object directly before rendering it!
    defineModelMatrix(shader, obj.Pos, obj.Scale)

    gl.BindVertexArray(obj.Geo.VertexObject)

    gl.Uniform3fv(gl.GetUniformLocation(shader, gl.Str("color\x00")), 1, &obj.Color[0])
    gl.Uniform3fv(gl.GetUniformLocation(shader, gl.Str("light\x00")), 1, &g_light.Pos[0])
    var isLighti int32 = 0
    if obj.IsLight {
        isLighti = 1
    }
    gl.Uniform1i(gl.GetUniformLocation(shader, gl.Str("isLight\x00")), isLighti)

    // And draw one panel.
    gl.DrawArrays(gl.TRIANGLES, 0, obj.Geo.VertexCount)

    gl.BindVertexArray(0)

}

func renderAllObjects(shader uint32) {
    for _,obj := range g_objects {
        renderObject(shader, obj)
    }
    renderObject(shader, g_light)
}

func renderFromCamera(shader uint32) {
    var fbo uint32 = 0
    if g_multisamplingEnabled {
        fbo = g_msFbo
    }

    gl.BindFramebuffer(gl.FRAMEBUFFER, fbo)
    gl.Enable(gl.DEPTH_TEST)
    // Nice blueish background
    gl.ClearColor(135.0/255.,206.0/255.,235.0/255., 1.0)

    gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
    gl.Viewport(0, 0, g_WindowWidth, g_WindowHeight)

    gl.UseProgram(shader)

    // projects a vertex into the light's framebuffer for shadow mapping!
    gl.UniformMatrix4fv(gl.GetUniformLocation(shader, gl.Str("lightProjectionMat\x00")), 1, false, &g_lightViewProjectionMat[0])
    // Light space depth
    gl.ActiveTexture(gl.TEXTURE0);
    gl.BindTexture(gl.TEXTURE_2D, g_lightColorTex);
    gl.Uniform1i(gl.GetUniformLocation(shader, gl.Str("lightDepthTex\x00")), 0)
    var shadowEnabledI int32 = 0
    if g_shadowEnabled {
        shadowEnabledI = 1
    }
    gl.Uniform1i(gl.GetUniformLocation(shader, gl.Str("shadowEnabled\x00")), shadowEnabledI)

    defineMatrices(shader)

    renderAllObjects(shader)

    gl.UseProgram(0)

    if g_multisamplingEnabled {
        gl.BindFramebuffer(gl.DRAW_FRAMEBUFFER, 0);   // Make sure no FBO is set as the draw framebuffer
        gl.BindFramebuffer(gl.READ_FRAMEBUFFER, g_msFbo); // Make sure your multisampled FBO is the read framebuffer
        gl.DrawBuffer(gl.BACK);                       // Set the back buffer as the draw buffer
        gl.BlitFramebuffer(0, 0, g_WindowWidth, g_WindowHeight, 0, 0, g_WindowWidth, g_WindowHeight, gl.COLOR_BUFFER_BIT, gl.NEAREST);
    }
}

// Render from light's perspective
func renderFromLight(shader uint32) {
    gl.CullFace(gl.FRONT)
    gl.BindFramebuffer(gl.FRAMEBUFFER, g_lightFbo)
    gl.Enable(gl.DEPTH_TEST)
    gl.ClearColor(0,0,0,1)

    gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
    gl.Viewport(0, 0, g_smWidth, g_smHeight)

    gl.UseProgram(shader)
    defineLightMatrices(shader)

    renderAllObjects(shader)

    gl.UseProgram(0)
    gl.CullFace(gl.BACK)
}

func blurTexture(inTex, outTex uint32, vertical, kernelSize int32) {
    gl.UseProgram(g_boxBlurCompute)
    gl.BindImageTexture(0, inTex, 0, false, 0, gl.READ_ONLY, gl.RG32F)
    gl.BindImageTexture(1, outTex, 0, false, 0, gl.WRITE_ONLY, gl.RG32F)
    gl.Uniform1i(gl.GetUniformLocation(g_boxBlurCompute, gl.Str("width\x00")), g_smWidth);
    gl.Uniform1i(gl.GetUniformLocation(g_boxBlurCompute, gl.Str("height\x00")), g_smHeight)
    gl.Uniform1i(gl.GetUniformLocation(g_boxBlurCompute, gl.Str("blurVertical\x00")), vertical)
    gl.Uniform1i(gl.GetUniformLocation(g_boxBlurCompute, gl.Str("kernelSize\x00")), kernelSize)

    gl.DispatchCompute(uint32(g_smWidth/240), 1, 1)
    gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)
    gl.UseProgram(0)
}

func render(window *glfw.Window) {

    if g_shadowEnabled {
        renderFromLight(g_varianceDepthShader)
        blurTexture(g_lightColorTex, g_lightColorTexBlur1, 1, g_blurKernelSize)
        blurTexture(g_lightColorTexBlur1, g_lightColorTex, 0, g_blurKernelSize)
        blurTexture(g_lightColorTex, g_lightColorTexBlur1, 1, g_blurKernelSize)
        blurTexture(g_lightColorTexBlur1, g_lightColorTex, 0, g_blurKernelSize)
    }

    renderFromCamera(g_ShaderID)


}

// Callback method for a keyboard press
func cbKeyboard(window *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {

    // All changes come VERY easy now.
    if action == glfw.Press {
        switch key {
            // Close the Simulation.
            case glfw.KeyEscape, glfw.KeyQ:
                window.SetShouldClose(true)
            case glfw.KeyH:
                printHelp()
            case glfw.KeySpace:
            case glfw.KeyF1:
                g_fillMode += 1
                switch (g_fillMode%3) {
                    case 0:
                        gl.PolygonMode(gl.FRONT_AND_BACK, gl.FILL)
                    case 1:
                        gl.PolygonMode(gl.FRONT_AND_BACK, gl.LINE)
                    case 2:
                        gl.PolygonMode(gl.FRONT_AND_BACK, gl.POINT)
                }
            case glfw.KeyF2:
                g_multisamplingEnabled = !g_multisamplingEnabled

                if g_multisamplingEnabled {
                    gl.Enable(gl.MULTISAMPLE)
                } else {
                    gl.Disable(gl.MULTISAMPLE)
                }
            case glfw.KeyF3:
                g_shadowEnabled = !g_shadowEnabled
            case glfw.KeyUp:
                g_light.Pos = g_light.Pos.Add(mgl32.Vec3{0,1.0,0})
            case glfw.KeyDown:
                g_light.Pos = g_light.Pos.Add(mgl32.Vec3{0,-1.0,0})
            case glfw.KeyLeft:
                if g_blurKernelSize >= 2 {
                    g_blurKernelSize -= 2
                }
            case glfw.KeyRight:
                g_blurKernelSize += 2
        }
    }

}

// see: https://github.com/go-gl/glfw/blob/master/v3.2/glfw/input.go
func cbMouseScroll(window *glfw.Window, xpos, ypos float64) {
    UpdateMouseScroll(xpos, ypos)
}

func cbMouseButton(window *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
    UpdateMouseButton(button, action, mods)
}

func cbCursorPos(window *glfw.Window, xpos, ypos float64) {
    UpdateCursorPos(xpos, ypos)
}


// Register all needed callbacks
func registerCallBacks (window *glfw.Window) {
    window.SetKeyCallback(cbKeyboard)
    window.SetScrollCallback(cbMouseScroll)
    window.SetMouseButtonCallback(cbMouseButton)
    window.SetCursorPosCallback(cbCursorPos)
}


func displayFPS(window *glfw.Window) {
    currentTime := glfw.GetTime()
    g_timeSum += float32(currentTime - g_lastCallTime)


    if g_frameCount%60 == 0 {
        g_fps = float32(1.0) / (g_timeSum/60.0)
        g_timeSum = 0.0

        s := fmt.Sprintf("FPS: %.1f", g_fps)
        window.SetTitle(s)
    }

    g_lastCallTime = currentTime
    g_frameCount += 1

}

// Mainloop for graphics updates and object animation
func mainLoop (window *glfw.Window) {

    registerCallBacks(window)
    glfw.SwapInterval(0)
    gl.Enable(gl.MULTISAMPLE)

    for !window.ShouldClose() {

        displayFPS(window)

        // This actually renders everything.
        render(window)

        window.SwapBuffers()
        glfw.PollEvents()
    }

}

func main() {
    var err error = nil
    if err = glfw.Init(); err != nil {
        panic(err)
    }
    // Terminate as soon, as this the function is finished.
    defer glfw.Terminate()

    window, err := initGraphicContext()
    if err != nil {
        // Decision to panic or do something different is taken in the main
        // method and not in sub-functions
        panic(err)
    }

    g_ShaderID, err = NewProgram("vertexShader.vert", "fragmentShader.frag")
    if err != nil {
        panic(err)
    }

    g_varianceDepthShader, err = NewProgram("simple.vert", "depthShader.frag")
    if err != nil {
        panic(err)
    }

    g_boxBlurCompute, err = NewComputeProgram("movingBoxBlur.comp")
    if err != nil {
        panic(err)
    }


    g_objects = append(g_objects, CreateObject(CreateSurface(10),    mgl32.Vec3{0,0,0},   mgl32.Vec3{100,1,100}, mgl32.Vec3{0.53, 0.81, 0.92}, false))
    g_objects = append(g_objects, CreateObject(CreateUnitCube(10),   mgl32.Vec3{4,8,-5}, mgl32.Vec3{3,1,1},   mgl32.Vec3{0.21,0.41,1},      false))
    g_objects = append(g_objects, CreateObject(CreateUnitSphere(10), mgl32.Vec3{-2,4,-3}, mgl32.Vec3{3,3,3},   mgl32.Vec3{1,0.06,0.14},      false))

    g_light = CreateObject(CreateUnitSphere(10), mgl32.Vec3{3,16,0}, mgl32.Vec3{0.1,0.1,0.1}, mgl32.Vec3{1,1,0}, true)

    CreateFbo(&g_msFbo, &g_msColorTex, &g_msDepthTex, g_WindowWidth, g_WindowHeight, g_multisamplingEnabled)

    CreateLightFbo(&g_lightFbo, &g_lightColorTex, &g_lightDepthTex, g_smWidth, g_smHeight, false)

    CreateTexture(&g_lightColorTexBlur1, g_smWidth, g_smHeight, gl.RG32F, gl.RG, gl.FLOAT)


    mainLoop(window)

}
