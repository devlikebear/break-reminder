import SwiftUI

struct CircularProgressRing: View {
    let progress: Double
    let fillColor: Color
    let trackColor: Color
    let lineWidth: CGFloat

    init(progress: Double, fillColor: Color, trackColor: Color = Color(white: 0.2), lineWidth: CGFloat = 10) {
        self.progress = progress
        self.fillColor = fillColor
        self.trackColor = trackColor
        self.lineWidth = lineWidth
    }

    var body: some View {
        ZStack {
            Circle()
                .stroke(trackColor, lineWidth: lineWidth)

            Circle()
                .trim(from: 0, to: CGFloat(min(progress, 1.0)))
                .stroke(fillColor, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))
                .rotationEffect(.degrees(-90))
        }
    }
}
