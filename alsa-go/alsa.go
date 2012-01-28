// alsa package is the simple wrapper for C alsa binding library.
package alsa

// #include <alsa/asoundlib.h>
import "C"

import (
	"os"
	"unsafe"
	"fmt"
)

// Alsa stream type. Playback or capture.
type StreamType C.snd_pcm_stream_t

// Stream type constants.
const (
	// Playback stream
	StreamTypePlayback = C.SND_PCM_STREAM_PLAYBACK
	// Capture stream
	StreamTypeCapture = C.SND_PCM_STREAM_CAPTURE
)

// Sample type.
type SampleFormat C.snd_pcm_format_t

const (
	// Unknown
	SampleFormatUnknown = C.SND_PCM_FORMAT_UNKNOWN
	// Signed 8 bit
	SampleFormatS8 = C.SND_PCM_FORMAT_S8
	// Unsigned 8 bit
	SampleFormatU8 = C.SND_PCM_FORMAT_U8
	// Signed 16 bit Little Endian
	SampleFormatS16LE = C.SND_PCM_FORMAT_S16_LE
	// Signed 16 bit Big Endian
	SampleFormatS16BE = C.SND_PCM_FORMAT_S16_BE
	// Unsigned 16 bit Little Endian
	SampleFormatU16LE = C.SND_PCM_FORMAT_U16_LE
	// Unsigned 16 bit Big Endian
	SampleFormatU16BE = C.SND_PCM_FORMAT_U16_BE
	// Signed 24 bit Little Endian using low three bytes in 32-bit word
	SampleFormatS24LE = C.SND_PCM_FORMAT_S24_LE
	// Signed 24 bit Big Endian using low three bytes in 32-bit word
	SampleFormatS24BE = C.SND_PCM_FORMAT_S24_BE
	// Unsigned 24 bit Little Endian using low three bytes in 32-bit word
	SampleFormatU24LE = C.SND_PCM_FORMAT_U24_LE
	// Unsigned 24 bit Big Endian using low three bytes in 32-bit word
	SampleFormatU24BE = C.SND_PCM_FORMAT_U24_BE
	// Signed 32 bit Little Endian
	SampleFormatS32LE = C.SND_PCM_FORMAT_S32_LE
	// Signed 32 bit Big Endian
	SampleFormatS32BE = C.SND_PCM_FORMAT_S32_BE
	// Unsigned 32 bit Little Endian
	SampleFormatU32LE = C.SND_PCM_FORMAT_U32_LE
	// Unsigned 32 bit Big Endian
	SampleFormatU32BE = C.SND_PCM_FORMAT_U32_BE
	// Signed 24bit Little Endian in 3bytes format
	SampleFormatS24_3LE = C.SND_PCM_FORMAT_S24_3LE
	// Signed 24bit Big Endian in 3bytes format
	SampleFormatS24_3BE = C.SND_PCM_FORMAT_S24_3BE
	// Unsigned 24bit Little Endian in 3bytes format
	SampleFormatU24_3LE = C.SND_PCM_FORMAT_U24_3LE
	// Unsigned 24bit Big Endian in 3bytes format
	SampleFormatU24_3BE = C.SND_PCM_FORMAT_U24_3BE

	/*
	 SND_PCM_FORMAT_FLOAT_LE 	Float 32 bit Little Endian, Range -1.0 to 1.0
	 SND_PCM_FORMAT_FLOAT_BE 	Float 32 bit Big Endian, Range -1.0 to 1.0
	 SND_PCM_FORMAT_FLOAT64_LE 	Float 64 bit Little Endian, Range -1.0 to 1.0
	 SND_PCM_FORMAT_FLOAT64_BE 	Float 64 bit Big Endian, Range -1.0 to 1.0
	 SND_PCM_FORMAT_IEC958_SUBFRAME_LE 	IEC-958 Little Endian
	 SND_PCM_FORMAT_IEC958_SUBFRAME_BE 	IEC-958 Big Endian
	 SND_PCM_FORMAT_MU_LAW 	Mu-Law
	 SND_PCM_FORMAT_A_LAW 	A-Law
	 SND_PCM_FORMAT_IMA_ADPCM 	Ima-ADPCM
	 SND_PCM_FORMAT_MPEG 	MPEG
	 SND_PCM_FORMAT_GSM 	GSM
	 SND_PCM_FORMAT_SPECIAL 	Special
	 SND_PCM_FORMAT_S20_3LE 	Signed 20bit Little Endian in 3bytes format
	 SND_PCM_FORMAT_S20_3BE 	Signed 20bit Big Endian in 3bytes format
	 SND_PCM_FORMAT_U20_3LE 	Unsigned 20bit Little Endian in 3bytes format
	 SND_PCM_FORMAT_U20_3BE 	Unsigned 20bit Big Endian in 3bytes format
	 SND_PCM_FORMAT_S18_3LE 	Signed 18bit Little Endian in 3bytes format
	 SND_PCM_FORMAT_S18_3BE 	Signed 18bit Big Endian in 3bytes format
	 SND_PCM_FORMAT_U18_3LE 	Unsigned 18bit Little Endian in 3bytes format
	 SND_PCM_FORMAT_U18_3BE 	Unsigned 18bit Big Endian in 3bytes format
	 SND_PCM_FORMAT_S16 	Signed 16 bit CPU endian
	 SND_PCM_FORMAT_U16 	Unsigned 16 bit CPU endian
	 SND_PCM_FORMAT_S24 	Signed 24 bit CPU endian
	 SND_PCM_FORMAT_U24 	Unsigned 24 bit CPU endian
	 SND_PCM_FORMAT_S32 	Signed 32 bit CPU endian
	 SND_PCM_FORMAT_U32 	Unsigned 32 bit CPU endian
	 SND_PCM_FORMAT_FLOAT 	Float 32 bit CPU endian
	 SND_PCM_FORMAT_FLOAT64 	Float 64 bit CPU endian
	 SND_PCM_FORMAT_IEC958_SUBFRAME 	IEC-958 CPU Endian 
	*/
)

// Open mode constants.
const (
	ModeBlock    = 0
	ModeNonblock = C.SND_PCM_NONBLOCK
	ModeAsync    = C.SND_PCM_ASYNC
)

// Handle represents ALSA stream handler.
type Handle struct {
	cHandle *C.snd_pcm_t
	// Used samples format (size, endianness, signed).
	SampleFormat SampleFormat
	// Sample rate in Hz. Usual 44100.
	SampleRate int
	// Channels in the stream. 2 for stereo.
	Channels   int
	Periods    int
	Buffersize C.snd_pcm_uframes_t
}

// New returns newly initialized ALSA handler.
func New() *Handle {
	handler := new(Handle)

	return handler
}

// Open opens a stream.
func (handle *Handle) Open(device string, streamType StreamType, mode int) os.Error {
	cDevice := C.CString(device)
	defer C.free(unsafe.Pointer(cDevice))

	err := C.snd_pcm_open(&(handle.cHandle), cDevice,
		_Ctypedef_snd_pcm_stream_t(streamType),
		_Ctype_int(mode))

	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot open audio device '%s'. %s",
			device, strError(err)))
	}

	return nil
}

// ApplyHwParams changes ALSA hardware parameters for the current stream.
func (handle *Handle) ApplyHwParams() os.Error {
	var cHwParams *C.snd_pcm_hw_params_t

	err := C.snd_pcm_hw_params_malloc(&cHwParams)
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot allocate hardware parameter structure. %s",
			strError(err)))
	}

	err = C.snd_pcm_hw_params_any(handle.cHandle, cHwParams)
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot initialize hardware parameter structure. %s",
			strError(err)))
	}

	err = C.snd_pcm_hw_params_set_access(handle.cHandle, cHwParams, C.SND_PCM_ACCESS_RW_INTERLEAVED)
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot set access type. %s",
			strError(err)))
	}

	err = C.snd_pcm_hw_params_set_format(handle.cHandle, cHwParams, _Ctypedef_snd_pcm_format_t(handle.SampleFormat))
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot set sample format. %s",
			strError(err)))
	}

	var cSampleRate _Ctype_uint = _Ctype_uint(handle.SampleRate)
	err = C.snd_pcm_hw_params_set_rate_near(handle.cHandle, cHwParams, &cSampleRate, nil)
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot set sample rate. %s",
			strError(err)))
	}

	err = C.snd_pcm_hw_params_set_channels(handle.cHandle, cHwParams, _Ctype_uint(handle.Channels))
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot set number of channels. %s",
			strError(err)))
	}

	/*
	    // Set number of periods. Periods used to be called fragments.
	    err = C.snd_pcm_hw_params_set_periods(handle.cHandle, cHwParams, _Ctype_uint(handle.Periods), 0)
		if err < 0 {
			return os.NewError(fmt.Sprintf("Cannot set number of periods. %s",
				strError(err)))
	    }

	    // Set buffer size (in frames). The resulting latency is given by
	    // latency = periodsize * periods / (rate * bytes_per_frame)
	    //err = C.snd_pcm_hw_params_set_buffer_size(handle.cHandle, cHwParams, (periodsize * periods)>>2)
		//if err < 0 {
		//	return os.NewError(fmt.Sprintf("Cannot set buffersize. %s",
		//		strError(err)))
	    //}

		//var cBufferSize *C.snd_pcm_uframes_t
		//cBufferSize = _Ctype_snd_pcm_uframes_t(_Ctype_uint(handle.Periods) * _Ctype_uint(handle.Periodsize))>>2
	    err = C.snd_pcm_hw_params_set_buffer_size_near(handle.cHandle, cHwParams, &handle.Buffersize)
		if err < 0 {
			return os.NewError(fmt.Sprintf("Cannot set buffersize. %s",
				strError(err)))
	    }
	*/

	// Drain current data and make sure we aren't underrun.
	C.snd_pcm_drain(handle.cHandle)

	err = C.snd_pcm_hw_params(handle.cHandle, cHwParams)
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot set hardware parameters. %s",
			strError(err)))
	}

	C.snd_pcm_hw_params_free(cHwParams)

	return nil
}

func GetDevices() {

	var hints **unsafe.Pointer

	/* Enumerate sound devices */
	err := C.snd_device_name_hint(-1, C.CString("pcm"), hints)
	if err != 0 {
		fmt.Println(err)
	   return
	}
	fmt.Println(hints)

	/*var n *unsafe.Pointer
	n = *hints

	for *n != nil {

		var name *C.char
		name = C.snd_device_name_get_hint(*n, C.CString("NAME"))

		if name != nil {
		    //Copy name to another buffer and then free it
		    fmt.Println(name)
		}
		*n = unsafe.Pointer(int(*n)+1)
	}

	// Free hint buffer too
	C.snd_device_name_free_hint(*hints)*/

}

func SetAlsaMasterVolume(volume int) {
    var min, max _Ctype_long
    var handle *C.snd_mixer_t
    var sid *C.snd_mixer_selem_id_t
    card := "default"
    selem_name := "Master"

    C.snd_mixer_open(&handle, 0)
    C.snd_mixer_attach(handle, C.CString(card))
    C.snd_mixer_selem_register(handle, nil, nil)
    C.snd_mixer_load(handle)

    //C.snd_mixer_selem_id_alloca(&sid)
    C.snd_mixer_selem_id_set_index(sid, 0)
    C.snd_mixer_selem_id_set_name(sid, C.CString(selem_name))
    var elem *C.snd_mixer_elem_t 
	elem = C.snd_mixer_find_selem(handle, sid)

    C.snd_mixer_selem_get_playback_volume_range(elem, &min, &max)
    C.snd_mixer_selem_set_playback_volume_all(elem, _Ctype_long(volume * int(max) / 128))

    C.snd_mixer_close(handle)
}

func (handle *Handle) Drain() os.Error {

	err := C.snd_pcm_drain(handle.cHandle)
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot drain stream. %s",
			strError(err)))
	}
	return nil

}

func (handle *Handle) Drop() os.Error {

	err := C.snd_pcm_drop(handle.cHandle)
	if err < 0 {
		return os.NewError(fmt.Sprintf("Cannot drop stream. %s",
			strError(err)))
	}
	return nil
		
}

func (handle *Handle) MaxSampleRate() (int, os.Error) {

	var cHwParams *C.snd_pcm_hw_params_t

	err := C.snd_pcm_hw_params_malloc(&cHwParams)
	if err < 0 {
		return 0, os.NewError(fmt.Sprintf("Cannot allocate hardware parameter structure. %s",
			strError(err)))
	}

	err = C.snd_pcm_hw_params_any(handle.cHandle, cHwParams)
	if err < 0 {
		return 0, os.NewError(fmt.Sprintf("Cannot initialize hardware parameter structure. %s",
			strError(err)))
	}

    err = C.snd_pcm_hw_params_set_rate_resample(handle.cHandle, cHwParams, 0);
	if err < 0 {
		return 0, os.NewError(fmt.Sprintf("Cannot restrict configuration space to contain only real hardware rates. %s",
			strError(err)))
	}

	var maxRate C.uint
	var dir C.int

	err = C.snd_pcm_hw_params_get_rate_max(cHwParams, &maxRate, &dir)
	if err < 0 {
		return 0, os.NewError(fmt.Sprintf("Retrieving maximum samplerate failed. %s", err))
	}

	C.snd_pcm_hw_params_free(cHwParams)

	return int(maxRate), nil

}

// Delay returns the numbers of frames between the time that a frame that 
// is written to the PCM stream and it to be actually audible.
func (handle *Handle) Delay() (int, os.Error) {
	var delay C.snd_pcm_sframes_t
	err := C.snd_pcm_delay(handle.cHandle, &delay)
	if err < 0 {
		return 0, os.NewError(fmt.Sprintf("Retrieving delay failed. %s", strError(_Ctype_int(delay))))
	}

	return int(_Ctype_int(delay)), nil
}

func (handle *Handle) SkipFrames(frames int) (int, os.Error) {

	// Get safe count of frames which can be forwarded.
	var framesForwardable C.snd_pcm_sframes_t
	framesForwardable = C.snd_pcm_forwardable(handle.cHandle)  
	if framesForwardable < 0 {
		return 0, os.NewError(fmt.Sprintf("Retrieving forwardable frames failed. %s", strError(_Ctype_int(framesForwardable))))
	}

	if int(_Ctype_int(framesForwardable)) < frames {
		frames = int(_Ctype_int(framesForwardable))
	}

	// Move application frame position forward.
	var framesForwarded C.snd_pcm_sframes_t
	framesForwarded = C.snd_pcm_forward(handle.cHandle, C.snd_pcm_uframes_t(frames)) 	
	if framesForwarded < 0 {
		return 0, os.NewError(fmt.Sprintf("Cannot forward frames. %s", strError(_Ctype_int(framesForwarded))))
	}

	return int(_Ctype_int(framesForwarded)), nil
}

// Wait waits till buffer will be free for some new portion of data or
// delay time is runs out.
// true ok value means that PCM stream is ready for I/O, false -- timeout occured.
func (handle *Handle) Wait(maxDelay int) (ok bool, err os.Error) {
	res, err := C.snd_pcm_wait(handle.cHandle, _Ctype_int(maxDelay))
	if err != nil {
		return false, os.NewError(fmt.Sprintf("Pool failed. %s", err))
	}

	return res > 0, nil
}

// AvailUpdate returns number of bytes ready to be read/written.
func (handle *Handle) AvailUpdate() (freeBytes int, err os.Error) {
	frames := C.snd_pcm_avail_update(handle.cHandle)
	if frames < 0 {
		return 0, os.NewError(fmt.Sprintf("Retriving free buffer size failed. %s", strError(_Ctype_int(frames))))
	}

	return int(frames) * handle.FrameSize(), nil
}

// Write writes given PCM data.
// Returns wrote value is total bytes was written.
func (handle *Handle) Write(buf []byte) (wrote int, err os.Error) {
	frames := len(buf) / handle.SampleSize() / handle.Channels
	w := C.snd_pcm_writei(handle.cHandle, unsafe.Pointer(&buf[0]), _Ctypedef_snd_pcm_uframes_t(frames))

	// Underrun? Retry.
	if w == -C.EPIPE {
		fmt.Sprintf("Underrun: %s", strError(_Ctype_int(w)))
		C.snd_pcm_prepare(handle.cHandle)
		w = C.snd_pcm_writei(handle.cHandle, unsafe.Pointer(&buf[0]), _Ctypedef_snd_pcm_uframes_t(frames))
	}

	if w < 0 {
		return 0, os.NewError(fmt.Sprintf("Write failed. %s", strError(_Ctype_int(w))))
	}

	wrote = int(w)
	wrote *= handle.FrameSize()

	return wrote, nil
}

// Pause PCM.
func (handle *Handle) Pause() os.Error {
	err := C.snd_pcm_pause(handle.cHandle, 1)
	if err != 0 {
		return os.NewError(fmt.Sprintf("Pause failed. %s", strError(err)))
	}

	return nil
}

// Unpause PCM.
func (handle *Handle) Unpause() os.Error {
	err := C.snd_pcm_pause(handle.cHandle, 0)
	if err != 0 {
		return os.NewError(fmt.Sprintf("Unpause failed. %s", strError(err)))
	}

	return nil
}

// Close closes stream and release the handler.
func (handle *Handle) Close() {
	C.snd_pcm_close(handle.cHandle)
}

// SampleSize returns one sample size in bytes.
func (handle *Handle) SampleSize() int {
	switch handle.SampleFormat {
	case SampleFormatS8, SampleFormatU8:
		return 1
	case SampleFormatS16LE, SampleFormatS16BE,
		SampleFormatU16LE, SampleFormatU16BE:
		return 2
	case SampleFormatS24_3LE, SampleFormatS24_3BE,
		SampleFormatU24_3LE, SampleFormatU24_3BE:
		return 3
	case SampleFormatS24LE, SampleFormatS24BE,
		SampleFormatU24LE, SampleFormatU24BE,
		SampleFormatS32LE, SampleFormatS32BE,
		SampleFormatU32LE, SampleFormatU32BE:
		return 4
	}

	return 1
}

// FrameSize returns size of one frame in bytes.
func (handle *Handle) FrameSize() int {
	return handle.SampleSize() * handle.Channels
}

// strError retruns string description of ALSA error by its code.
func strError(err _Ctype_int) string {
	cErrMsg := C.snd_strerror(err)

	return C.GoString(cErrMsg)
}
