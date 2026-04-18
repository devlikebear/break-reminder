import SwiftUI

struct ConfettiParticle: Identifiable {
    let id = UUID()
    let x: CGFloat         // Start x position (0...1)
    let delay: TimeInterval
    let duration: TimeInterval
    let color: Color
    let rotationSpeed: Double

    static func random(colors: [Color]) -> ConfettiParticle {
        ConfettiParticle(
            x: .random(in: 0...1),
            delay: .random(in: 0...0.5),
            duration: .random(in: 1.5...3.0),
            color: colors.randomElement() ?? .green,
            rotationSpeed: .random(in: 0.5...2.0)
        )
    }
}

struct ConfettiView: View {
    let particles: [ConfettiParticle]
    let isActive: Bool

    static func generate(count: Int, colors: [Color]) -> [ConfettiParticle] {
        (0..<count).map { _ in ConfettiParticle.random(colors: colors) }
    }

    var body: some View {
        if isActive {
            TimelineView(.animation) { timeline in
                Canvas { context, size in
                    let elapsed = timeline.date.timeIntervalSinceReferenceDate
                        .truncatingRemainder(dividingBy: 3.5)

                    for particle in particles {
                        let t = max(0, elapsed - particle.delay)
                        guard t < particle.duration else { continue }
                        let progress = t / particle.duration

                        let x = particle.x * size.width + sin(t * 3) * 20
                        let y = progress * (size.height + 40)
                        let rotation = t * particle.rotationSpeed * .pi * 2

                        var transform = CGAffineTransform(translationX: x, y: y)
                        transform = transform.rotated(by: rotation)

                        let rect = CGRect(x: -4, y: -6, width: 8, height: 12).applying(transform)
                        context.fill(
                            Path(rect),
                            with: .color(particle.color.opacity(1.0 - progress))
                        )
                    }
                }
            }
            .allowsHitTesting(false)
        }
    }
}
