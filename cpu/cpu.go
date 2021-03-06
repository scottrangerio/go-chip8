package cpu

import (
	"math/rand"
	"time"

	"github.com/scottrangerio/go-chip8/cpu/opcode"
	"github.com/scottrangerio/go-chip8/display"
	m "github.com/scottrangerio/go-chip8/memory"
	"github.com/scottrangerio/go-chip8/sprites"
)

// LoadRom loads a rom into memory
func (c *CPU) LoadRom(d []byte) {
	s := 0x200

	c.memory.WriteBytesAt(d, s)
}

func (c *CPU) getOpcode() opcode.Opcode {
	return opcode.NewOpcode(c.memory.ReadByteAt(int(c.pc)), c.memory.ReadByteAt(int(c.pc+1)))
}

type memory interface {
	WriteBytesAt(b []byte, off int)
	ReadBytesAt(b []byte, off int)
	WriteByteAt(b byte, off int)
	ReadByteAt(off int) byte
}

// CPU represents a CHIP-8 CPU
type CPU struct {
	v      [16]byte
	sp     byte
	pc     uint16
	i      uint16
	stack  [16]uint16
	sound  byte
	timer  byte
	memory memory
	st     byte
	dt     byte
}

// NewCPU creates and initializes a new CPU
func NewCPU() *CPU {
	cpu := &CPU{
		pc:     0x200,
		memory: new(m.Memory),
	}

	for i, s := range sprites.Sprites {
		cpu.memory.WriteBytesAt(s[:], i+(i*4))
	}

	return cpu
}

// Run runs the emulator
func (c *CPU) Run(done chan struct{}, kb map[byte]bool) {
	d := new(display.Display)
	d.Init()
	defer d.Close()

	for {
		select {
		case <-done:
			return
		default:
			op := c.getOpcode()

			if c.dt > 0 {
				c.dt--
			}

			if c.st > 0 {
				c.st--
			}

			switch op.LeadByte() {
			case 0x0:
				c.pc = c.stack[c.sp]
				c.sp--
				c.pc += 2
			case 0x1:
				c.pc = op.NNN()
			case 0x2:
				c.sp++
				c.stack[c.sp] = c.pc
				c.pc = op.NNN()
			case 0x3:
				x := op.X()
				if c.v[x] == op.KK() {
					c.pc += 2
				}
				c.pc += 2
			case 0x4:
				x := op.X()
				if c.v[x] != op.KK() {
					c.pc += 2
				}
				c.pc += 2
			case 0x6:
				x := op.X()
				c.v[x] = op.KK()
				c.pc += 2
			case 0x7:
				x := op.X()
				c.v[x] = c.v[x] + op.KK()
				c.pc += 2
			case 0x8:
				switch op.N() {
				case 0x0:
					x := op.X()
					y := op.Y()
					c.v[x] = c.v[y]
					c.pc += 2
				case 0x1:
					x := op.X()
					y := op.Y()
					r := c.v[x] | c.v[y]
					c.v[x] = r
					c.pc += 2
				case 0x2:
					x := op.X()
					y := op.Y()
					r := c.v[x] & c.v[y]
					c.v[x] = r
					c.pc += 2
				case 0x3:
					x := op.X()
					y := op.Y()
					r := c.v[x] ^ c.v[y]
					c.v[x] = r
					c.pc += 2
				case 0x4:
					x := op.X()
					y := op.Y()
					r := uint16(c.v[x]) + uint16(c.v[y])
					if r > 0xFF {
						c.v[0xF] = 1
					} else {
						c.v[0xF] = 0
					}
					c.v[x] = byte(r | 0x00)
					c.pc += 2
				case 0x5:
					x := op.X()
					y := op.Y()
					if c.v[x] > c.v[y] {
						c.v[0xF] = 1
					} else {
						c.v[0xF] = 0
					}
					c.v[x] = c.v[x] - c.v[y]
					c.pc += 2
				case 0x6:
					x := op.X()
					c.v[0xF] = c.v[x] & 0x01
					c.v[x] /= 2
					c.pc += 2
				default:
					time.Sleep(1 * time.Second)
					return
				}
			case 0xA:
				c.i = op.NNN()
				c.pc += 2
			case 0xC:
				x := op.X()
				rand.Seed(time.Now().Unix())
				r := rand.Intn(255)
				c.v[x] = byte(r) & op.KK()
				c.pc += 2
			case 0xD:
				x := c.v[op.X()]
				y := c.v[op.Y()]
				n := uint16(op.N())

				b := make([]byte, n, n)
				c.memory.ReadBytesAt(b, int(c.i))

				d.DrawSprite(int(x), int(y), b)
				time.Sleep((1000 / 120) * time.Millisecond)
				c.pc += 2
			case 0xE:
				switch op.KK() {
				case 0x00A1:
					if !kb[c.v[op.X()]] {
						c.pc += 2
					}
					c.pc += 2
				case 0x009E:
					if kb[c.v[op.X()]] {
						c.pc += 2
					}
					c.pc += 2
				}
			case 0xF:
				switch op.KK() {
				case 0x0007:
					x := op.X()
					c.v[x] = c.dt
					c.pc += 2
				case 0x0015:
					x := op.X()
					c.dt = c.v[x]
					c.pc += 2
				case 0x0018:
					x := op.X()
					c.st = c.v[x]
					c.pc += 2
				case 0x0029:
					x := op.X()
					v := c.v[x] * 0x05
					c.i = uint16(v)
					c.pc += 2
				case 0x0033:
					x := op.X()
					c.memory.WriteByteAt(c.v[x]/100, int(c.i))
					c.memory.WriteByteAt((c.v[x]/10)%10, int(c.i+1))
					c.memory.WriteByteAt((c.v[x]%100)%10, int(c.i+2))

					c.pc += 2
				case 0x0065:
					x := uint16(op.X())

					for i := uint16(0); i <= x; i++ {
						c.v[i] = c.memory.ReadByteAt(int(c.i + i))
					}
					c.pc += 2
				default:
					time.Sleep(1 * time.Second)
					return
				}
			default:
				time.Sleep(1 * time.Second)
				return
			}
		}

	}
}
