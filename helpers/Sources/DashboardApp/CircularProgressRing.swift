import SwiftUI

struct CircularProgressRing: View {
    let progress: Double
    let fillColor: Color
    let lineWidth: CGFloat

    init(progress: Double, fillColor: Color, lineWidth: CGFloat = 10) {
        self.progress = progress
        self.fillColor = fillColor
        self.lineWidth = lineWidth
    }

    var body: some View {
        ZStack {
            Circle()
                .stroke(Color(white: 0.2), lineWidth: lineWidth)

            Circle()
                .trim(from: 0, to: CGFloat(min(progress, 1.0)))
                .stroke(fillColor, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))
                .rotationEffect(.degrees(-90))
        }
    }
}
